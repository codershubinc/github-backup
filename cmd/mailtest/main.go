package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github-backup/mailer"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../../.env")

	from := os.Getenv("GMAIL_USER")
	pass := os.Getenv("GMAIL_APP_PASSWORD")
	to := os.Getenv("MAIL_TO")

	if from == "" || pass == "" || to == "" {
		log.Fatal("GMAIL_USER, GMAIL_APP_PASSWORD, and MAIL_TO must be set")
	}

	report := mailer.ReportData{
		Total:       42,
		Success:     40,
		FailCount:   2,
		FailedRepos: []string{"secret-infra", "legacy-monolith"},
		Location:    "repos/",
		Duration:    3*time.Minute + 27*time.Second,
		RunAt:       time.Now(),
	}

	subject := "⚠️ GitHub Backup — 2 repo(s) failed"
	fmt.Printf("Sending test HTML email from %s to %s...\n", from, to)
	err := mailer.SendHTMLEmail(from, pass, to, subject, mailer.BuildReportHTML(report))
	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}
	fmt.Println("Email sent successfully.")
}
