package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	binaryPrefix = "/cockroach-"
)

var (
	blackList  = map[string]struct{}{}
	gzipReader = new(gzip.Reader)

	linuxRE  = regexp.MustCompile(`/cockroach-.*\.linux-.*\.tgz`)
	darwinRE = regexp.MustCompile(`/cockroach-.*\.darwin-.*\.tgz`)
	// We made a mistake at some point and used tgz on windows.
	windowsRE = regexp.MustCompile(`/cockroach-.*\.windows-.*\.(tgz|zip)`)
	sourceRE  = regexp.MustCompile(`/cockroach-.*\.src\.tgz`)
)

func parseAllFilesInDir(dirName string) {
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if !file.Mode().IsRegular() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".gz") {
			continue
		}
		if err := readGzipFile(filepath.Join(dirName, file.Name())); err != nil {
			log.Printf("error adding %s: %v", file.Name(), err)
		}
	}
}

// Read the contents of a file. Ungzips and reads all, adding it to the running map.
func readGzipFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := gzipReader.Reset(f); err != nil {
		return err
	}

	scanner := bufio.NewScanner(gzipReader)
	for scanner.Scan() {
		if err := addLine(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

type dayEntry struct {
	date       string // field 1
	ip         string // field 5
	url        string // field 8
	statusCode string // field 9
}

// See http://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/AccessLogs.html#BasicDistributionFileFormat
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
func dayEntryFromLine(line string) (dayEntry, error) {
	parts := strings.Split(line, "\t")
	// Allow more than the documented number, things may get added.
	if len(parts) < 23 {
		return dayEntry{}, fmt.Errorf("not enough entries in file, found %d", len(parts))
	}

	return dayEntry{
		date:       parts[0],
		ip:         parts[4],
		url:        parts[7],
		statusCode: parts[8],
	}, nil
}

func (de dayEntry) shouldSkip() bool {
	// We only want full successful responses.
	// We do get 206 (partial content), various 3XX/4XX, and the occasional 000 which cloudfront
	// uses to mean that the client closed the connection before the response.
	if de.statusCode != "200" {
		return true
	}

	if !strings.HasPrefix(de.url, binaryPrefix) {
		return true
	}

	if _, ok := blackList[de.ip]; ok {
		return true
	}
	return false
}

func (de dayEntry) downloadType() int {
	if linuxRE.MatchString(de.url) {
		return linuxDownload
	} else if darwinRE.MatchString(de.url) {
		return darwinDownload
	} else if sourceRE.MatchString(de.url) {
		return sourceDownload
	} else if windowsRE.MatchString(de.url) {
		return windowsDownload
	} else {
		return unknownDownload
	}
}

// Process a single line from a file.
func addLine(line string) error {
	if len(line) == 0 {
		return nil
	}
	if line[0] == '#' {
		return nil
	}

	entry, err := dayEntryFromLine(line)
	if err != nil {
		return err
	}

	if entry.shouldSkip() {
		return nil
	}

	var daySummary *summary
	var ok bool
	if daySummary, ok = dayMap[entry.date]; !ok {
		daySummary = &summary{Date: entry.date}
		dayMap[entry.date] = daySummary
	}

	daySummary.Total++
	daySummary.TypeCounts[entry.downloadType()]++

	return nil
}
