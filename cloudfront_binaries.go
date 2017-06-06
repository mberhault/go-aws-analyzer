package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// AWS CloudFront log format:
// http://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/AccessLogs.html#BasicDistributionFileFormat
// 1 date
// 2 time
// 3 x-edge-location
// 4 sc-bytes
// 5 c-ip
// 6 cs-method
// 7 cs(Host)
// 8 cs-uri-stem
// 9 sc-status
// 10  cs(Referer)
// 11  cs(User-Agent)
// 12  cs-uri-query
// 13  cs(Cookie)
// 14  x-edge-result-type
// 15  x-edge-request-id
// 16  x-host-header
// 17  cs-protocol
// 18  cs-bytes
// 19  time-taken
// 20  x-forwarded-for
// 21  ssl-protocol
// 22  ssl-cipher
// 23  x-edge-response-result-type
// 24  cs-protocol-version

const (
	binaryPrefix = "/cockroach-"
	cfDateFormat = "2006-01-02"

	// Download types. We're using them in the template, so don't use iota.
	unknownDownload  = 0
	linuxDownload    = 1
	darwinDownload   = 2
	windowsDownload  = 3
	sourceDownload   = 4
	numDownloadTypes = 5
)

var (
	dirName       = flag.String("dir", "", "Directory containing gzipped aws logs")
	blackListFlag = flag.String("blacklist", "", "Comma-separated list of client IPs to ignore")

	linuxRE  = regexp.MustCompile(`/cockroach-.*\.linux-.*\.tgz`)
	darwinRE = regexp.MustCompile(`/cockroach-.*\.darwin-.*\.tgz`)
	// We made a mistake at some point and used tgz on windows.
	windowsRE = regexp.MustCompile(`/cockroach-.*\.windows-.*\.(tgz|zip)`)
	sourceRE  = regexp.MustCompile(`/cockroach-.*\.src\.tgz`)
)

func GenerateCloudFront() (string, []string) {
	cfd := NewCFDownloads()
	cfEntries := cfd.Run()
	buf := new(bytes.Buffer)

	// Per-day summaries.
	daily := cfEntries.ByInterval(time.Hour * 24)
	spec := RenderSpec{"Daily Downloads", cfDateFormat, daily, CFTitles(), CFEntryData, 10}

	dailyImage := "daily.png"
	RenderTemplate(buf, spec)
	if _, err := fmt.Fprintf(buf, `<img src="cid:%s" alt="spec.Title">`, dailyImage); err != nil {
		log.Fatalf("failed to write image tag for %s: %v", dailyImage, err)
	}
	spec.NumToDisplay = 60 // Two month
	RenderChartToFile(dailyImage, spec)

	// Per-week summaries.
	weekly := cfEntries.ByInterval(time.Hour * 24 * 7)
	spec = RenderSpec{"Weekly Downloads", cfDateFormat, weekly, CFTitles(), CFEntryData, 10}

	weeklyImage := "weekly.png"
	RenderTemplate(buf, spec)
	if _, err := fmt.Fprintf(buf, `<img src="cid:%s" alt="spec.Title">`, weeklyImage); err != nil {
		log.Fatalf("failed to write image tag for %s: %v", weeklyImage, err)
	}
	spec.NumToDisplay = 52 // One year.
	RenderChartToFile(weeklyImage, spec)

	return buf.String(), []string{dailyImage, weeklyImage}
}

type CFDownloads struct {
	dir        string
	blackList  map[string]struct{}
	gzipReader *gzip.Reader
	entries    *EntryManager
}

func NewCFDownloads() *CFDownloads {
	ret := &CFDownloads{
		dir:        *dirName,
		blackList:  make(map[string]struct{}),
		gzipReader: new(gzip.Reader),
		entries:    NewEntryManager(numDownloadTypes),
	}

	if len(*blackListFlag) > 0 {
		parts := strings.Split(*blackListFlag, ",")
		for _, p := range parts {
			ret.blackList[p] = struct{}{}
		}
	}

	return ret
}

func (cf *CFDownloads) Run() *EntryManager {
	cf.parseAllFilesInDir()
	cf.entries.Order()
	return cf.entries
}

func CFTitles() []string {
	return []string{"Total", "Linux", "Darwin", "Windows", "Source"}
}

func CFEntryData(de *DatedEntry) []float64 {
	ret := make([]float64, numDownloadTypes)
	for i := linuxDownload; i < numDownloadTypes; i++ {
		ret[0] += de.totals[i]
		ret[i] += de.totals[i]
	}
	return ret
}

func (cf *CFDownloads) parseAllFilesInDir() {
	files, err := ioutil.ReadDir(cf.dir)
	if err != nil {
		log.Fatalf("could not ReadDir(%q): %v", cf.dir, err)
	}

	for _, file := range files {
		if !file.Mode().IsRegular() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".gz") {
			continue
		}
		if err := cf.readGzipFile(filepath.Join(cf.dir, file.Name())); err != nil {
			log.Printf("error adding %s: %v", file.Name(), err)
		}
	}
}

// Read the contents of a file. Ungzips and reads all, adding it to the running map.
func (cf *CFDownloads) readGzipFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := cf.gzipReader.Reset(f); err != nil {
		return err
	}

	scanner := bufio.NewScanner(cf.gzipReader)
	for scanner.Scan() {
		if err := cf.parseOneLine(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func downloadType(url string) int {
	if linuxRE.MatchString(url) {
		return linuxDownload
	} else if darwinRE.MatchString(url) {
		return darwinDownload
	} else if sourceRE.MatchString(url) {
		return sourceDownload
	} else if windowsRE.MatchString(url) {
		return windowsDownload
	} else {
		return unknownDownload
	}
}

// Process a single line from a file.
func (cf *CFDownloads) parseOneLine(line string) error {
	if len(line) == 0 {
		return nil
	}
	if line[0] == '#' {
		return nil
	}

	parts := strings.Split(line, "\t")
	// Allow more than the documented number, things may get added.
	if len(parts) < 23 {
		return fmt.Errorf("only %d entries in line %s", len(parts), line)
	}

	date, ip, method, url, statusCode := parts[0], parts[4], parts[5], parts[7], parts[8]

	if method != "GET" {
		return nil
	}
	// We only want full successful responses.
	// We do get 206 (partial content), various 3XX/4XX, and the occasional 000 which cloudfront
	// uses to mean that the client closed the connection before the response.
	if statusCode != "200" {
		return nil
	}
	if !strings.HasPrefix(url, binaryPrefix) {
		return nil
	}
	if _, ok := cf.blackList[ip]; ok {
		return nil
	}

	parsedDate, err := time.Parse(cfDateFormat, date)
	if err != nil {
		return err
	}

	dt := downloadType(url)

	cf.entries.AddSample(parsedDate, dt, 1)

	return nil
}
