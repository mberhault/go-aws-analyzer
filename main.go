package main

import (
	"flag"
	"log"
	"os"
	"sort"
	"strings"
)

const (
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
	htmlFile      = flag.String("summary-html", "summary.html", "Filename for html summary")
	dailyPng      = flag.String("daily-png", "daily.png", "Filename for daily png chart")
	weeklyPng     = flag.String("weekly-png", "weekly.png", "Filename for weekly png chart")
)

func main() {
	flag.Parse()

	if len(*blackListFlag) > 0 {
		parts := strings.Split(*blackListFlag, ",")
		for _, p := range parts {
			blackList[p] = struct{}{}
		}
	}

	// Parse everything. This may crash.
	parseAllFilesInDir(*dirName)

	// Convert hash map to sorted slice.
	buildOrdered()

	// Write the chart.
	writeChart(*dailyPng, daily[len(daily)-62:])
	writeChart(*weeklyPng, weekly)

	// Write the HTML summary.

	f, err := os.Create(*htmlFile)
	if err != nil {
		log.Fatalf("could not create summary files: %v", err)
	}
	defer f.Close()

	// We need to reverser the data first, templates don't have for loops.
	sort.Sort(sort.Reverse(daily))
	sort.Sort(sort.Reverse(weekly))

	writeHTML(f, "Daily downloads", daily[0:10])
	writeHTML(f, "Weekly downloads", weekly[0:10])
}
