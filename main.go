package main

import (
	"flag"
	"log"
	"time"

	gomail "gopkg.in/gomail.v2"
)

var (
	emailFrom     = flag.String("email-from", "", "SMTP server from address")
	emailTo       = flag.String("email-to", "", "SMTP server to address")
	emailHost     = flag.String("email-host", "", "SMTP server name")
	emailPort     = flag.Int("email-port", 587, "SMTP server port")
	emailUser     = flag.String("email-user", "", "SMTP server username")
	emailPassword = flag.String("email-password", "", "SMTP server password")
)

func main() {
	flag.Parse()

	body, files := GenerateCloudFront()

	// Craft the email.
	m := gomail.NewMessage()
	m.SetHeader("From", *emailFrom)
	m.SetHeader("To", *emailTo)
	m.SetHeader("Subject", time.Now().Format("Binary downloads 2006-01-02"))
	m.SetBody("text/html", body)
	for _, f := range files {
		m.Embed(f)
	}

	// Send mail.
	d := gomail.NewDialer(*emailHost, *emailPort, *emailUser, *emailPassword)
	if err := d.DialAndSend(m); err != nil {
		log.Fatalf("could not send email: %v", err)
	}
}
