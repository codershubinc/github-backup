package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github-backup/mailer"
	"github.com/joho/godotenv"
)

// Config
const (
	MaxConcurrency = 5
	BackupDir      = "repos"
)

type Repo struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func main() {
	// 1. Setup Env
	_ = godotenv.Load() // load .env if present; silently ignored if missing

	targetDir := os.Getenv("BACKUP_DIR")
	if targetDir == "" {
		targetDir = BackupDir
	}
	orgName := os.Getenv("ORG_NAME")
	
	// Email Env
	gmailUser := os.Getenv("GMAIL_USER")
	gmailPass := os.Getenv("GMAIL_APP_PASSWORD")
	mailTo := os.Getenv("MAIL_TO")

	fmt.Println("🚀 GitHub Backup Service Started")

	// 2. Fetch Repos via GH CLI
	fmt.Print("🔍 Fetching repository list via gh cli... ")
	args := []string{"repo", "list"}
	if orgName != "" {
		args = append(args, orgName)
	}
	args = append(args, "--limit", "4000", "--json", "name,url,owner")

	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		fatalMsg := fmt.Sprintf("Failed to list repos. Is 'gh' authenticated? Error: %v", err)
		mailer.SendEmail(gmailUser, gmailPass, mailTo, "❌ Backup Failed: Initial Fetch", fatalMsg)
		log.Fatal(fatalMsg)
	}
	fmt.Println("Done.")

	var repos []Repo
	if err := json.Unmarshal(output, &repos); err != nil {
		log.Fatalf("❌ Failed to parse JSON: %v", err)
	}

	fmt.Printf("📦 Found %d repositories. Backing up to: %s\n\n", len(repos), targetDir)
	os.MkdirAll(targetDir, 0755)

	startTime := time.Now()

	// 3. Concurrent Workers
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, MaxConcurrency)
	
	// Track stats safely
	var successCount, failCount int
	var mu sync.Mutex
	var failedRepos []string

	for _, repo := range repos {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(r Repo) {
			defer wg.Done()
			defer func() { <-semaphore }()
			finalPath := filepath.Join(targetDir, r.Name+".git")
			action := "CLONE"
			if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
				action = "UPDATE"
			}

			// Spinner logic
			stopSpinner := make(chan bool)
			go showSpinner(r.Name, action, stopSpinner)

			var err error
			if action == "UPDATE" {
				gitCmd := exec.Command("git", "remote", "update")
				gitCmd.Dir = finalPath
				err = gitCmd.Run()
			} else {
				ghCmd := exec.Command("gh", "repo", "clone", fmt.Sprintf("%s/%s", r.Owner.Login, r.Name), finalPath, "--", "--mirror")
				err = ghCmd.Run()
			}

			stopSpinner <- true

			mu.Lock()
			if err != nil {
				failCount++
				failedRepos = append(failedRepos, r.Name)
				fmt.Printf("\r❌ [%s] %s Failed: %v\n", action, r.Name, err)
			} else {
				successCount++
				fmt.Printf("\r✅ [%s] %s Complete\n", action, r.Name)
			}
			mu.Unlock()
		}(repo)
	}

	wg.Wait()
	duration := time.Since(startTime)
	fmt.Println("\n🎉 All backups complete.")

	// 4. Send Summary Email
	if gmailUser != "" && gmailPass != "" && mailTo != "" {
		fmt.Println("📧 Sending email report...")

		subject := "✅ GitHub Backup — All repos synced successfully"
		if failCount > 0 {
			subject = fmt.Sprintf("⚠️ GitHub Backup — %d repo(s) failed", failCount)
		}

		report := mailer.ReportData{
			Total:       len(repos),
			Success:     successCount,
			FailCount:   failCount,
			FailedRepos: failedRepos,
			Location:    targetDir,
			Duration:    duration,
			RunAt:       time.Now(),
		}
		report.Hostname, report.OS, report.Arch = mailer.MachineInfo()

		err := mailer.SendHTMLEmail(gmailUser, gmailPass, mailTo, subject, mailer.BuildReportHTML(report))
		if err != nil {
			fmt.Printf("❌ Failed to send email: %v\n", err)
		} else {
			fmt.Println("✅ Email sent successfully.")
		}
	} else {
		fmt.Println("ℹ️ Skipping email (Credentials not set).")
	}
}

func showSpinner(repoName, action string, stop chan bool) {
	// Simple check to see if we are in a terminal (very basic)
	// If output is redirected (like in systemd), we might want to skip this loop to avoid log spam.
	// For now, we keep it simple.
	i := 0
	for {
		select {
		case <-stop:
			return
		default:
			fmt.Printf("\r%s [%s] %s...", spinnerChars[i%len(spinnerChars)], action, repoName)
			time.Sleep(100 * time.Millisecond)
			i++
		}
	}
}

