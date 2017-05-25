package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	gomail "gopkg.in/gomail.v2"
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

	emailFrom     = flag.String("email-from", "", "SMTP server from address")
	emailTo       = flag.String("email-to", "", "SMTP server to address")
	emailHost     = flag.String("email-host", "", "SMTP server name")
	emailPort     = flag.Int("email-port", 587, "SMTP server port")
	emailUser     = flag.String("email-user", "", "SMTP server username")
	emailPassword = flag.String("email-password", "", "SMTP server password")
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

	f, err := os.Create(*htmlFile)
	if err != nil {
		log.Fatalf("could not create summary files: %v", err)
	}
	defer f.Close()

	// We need to reverse the data first, templates don't have for loops.
	sort.Sort(sort.Reverse(daily))
	sort.Sort(sort.Reverse(weekly))

	// Write the HTML summary.
	writeHTML(f, "Daily downloads", daily[0:10])
	writeHTML(f, "Weekly downloads", weekly[0:10])

	// Craft the email.
	m := gomail.NewMessage()
	m.SetHeader("From", *emailFrom)
	m.SetHeader("To", *emailTo)
	m.SetHeader("Subject", time.Now().Format("Binary downloads 2006-01-02"))
	m.Embed(*dailyPng)
	m.Embed(*weeklyPng)
	m.AddAlternativeWriter("text/html", func(w io.Writer) error {
		writeHTML(w, "Daily downloads", daily[0:10])
		fmt.Fprintf(w, `<img src="cid:%s">`, *dailyPng)
		writeHTML(w, "Weekly downloads", weekly[0:10])
		fmt.Fprintf(w, `<img src="cid:%s">`, *weeklyPng)
		return nil
	})

	// Send mail.
	d := gomail.NewDialer(*emailHost, *emailPort, *emailUser, *emailPassword)
	if err := d.DialAndSend(m); err != nil {
		log.Fatalf("could not send email: %v", err)
	}
}
