package mailer

import (
	"fmt"
	"html"
	"net/smtp"
	"os"
	"runtime"
	"strings"
	"time"
)

// ReportData holds backup run statistics used to render the HTML report email.
type ReportData struct {
	Total       int
	Success     int
	FailCount   int
	FailedRepos []string
	Location    string
	Duration    time.Duration
	RunAt       time.Time
	Hostname    string
	OS          string
	Arch        string
}

// MachineInfo populates Hostname, OS, and Arch from the current runtime.
func MachineInfo() (hostname, osName, arch string) {
	hostname, _ = os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return hostname, runtime.GOOS, runtime.GOARCH
}

// SendHTMLEmail sends an HTML email via Gmail SMTP.
func SendHTMLEmail(from, password, to, subject, htmlBody string) error {
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	msg := "From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
		htmlBody

	auth := smtp.PlainAuth("", from, password, smtpHost)
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(msg))
}

// SendEmail sends a plain-text fallback email via Gmail SMTP.
func SendEmail(from, password, to, subject, body string) error {
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	msg := "From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" +
		body

	auth := smtp.PlainAuth("", from, password, smtpHost)
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(msg))
}

// BuildReportHTML renders a GitHub-dark-themed HTML email body for the backup report.
func BuildReportHTML(d ReportData) string {
	badgeBg, badgeColor, badgeText := "#1a2e22", "#3fb950", "✅ All Backups Succeeded"
	if d.FailCount > 0 {
		badgeBg, badgeColor, badgeText = "#2d1b1b", "#f85149", "⚠️ Completed with Errors"
	}

	failCardBg, failCardBorder, failCardColor := "#21262d", "#30363d", "#8b949e"
	if d.FailCount > 0 {
		failCardBg, failCardBorder, failCardColor = "#2d1b1b", "#f85149", "#f85149"
	}

	successPct := 0
	if d.Total > 0 {
		successPct = int(float64(d.Success) / float64(d.Total) * 100)
	}

	failedSection := ""
	if len(d.FailedRepos) > 0 {
		var rows strings.Builder
		for i, name := range d.FailedRepos {
			rowBg := "#161b22"
			if i%2 == 0 {
				rowBg = "#0d1117"
			}
			rows.WriteString(fmt.Sprintf(`
				<tr style="background:%s;">
					<td style="padding:10px 16px;font-size:12px;color:#8b949e;border-bottom:1px solid #21262d;">%d</td>
					<td style="padding:10px 16px;font-size:13px;color:#ff7b72;font-family:'SFMono-Regular',Consolas,monospace;border-bottom:1px solid #21262d;">%s</td>
				</tr>`, rowBg, i+1, html.EscapeString(name)))
		}
		failedSection = fmt.Sprintf(`
		<tr>
			<td style="background:#161b22;padding:0 32px 24px;border-left:1px solid #30363d;border-right:1px solid #30363d;">
				<div style="font-size:13px;font-weight:600;color:#f0f6fc;margin-bottom:12px;">Failed Repositories</div>
				<table width="100%%" cellpadding="0" cellspacing="0" style="border-radius:6px;overflow:hidden;border:1px solid #30363d;width:100%%;">
					<tr style="background:#21262d;">
						<th style="padding:10px 16px;text-align:left;font-size:12px;color:#8b949e;font-weight:500;border-bottom:1px solid #30363d;">#</th>
						<th style="padding:10px 16px;text-align:left;font-size:12px;color:#8b949e;font-weight:500;border-bottom:1px solid #30363d;">Repository</th>
					</tr>
					%s
				</table>
			</td>
		</tr>`, rows.String())
	}

	runDate := d.RunAt.Format("Jan 2, 2006 · 15:04:05 MST")
	dur := d.Duration.Round(time.Second).String()
	year := d.RunAt.Format("2006")

	hostname := d.Hostname
	if hostname == "" {
		hostname, _, _ = MachineInfo()
	}
	osName := d.OS
	if osName == "" {
		_, osName, _ = MachineInfo()
	}
	arch := d.Arch
	if arch == "" {
		_, _, arch = MachineInfo()
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1.0">
  <title>GitHub Backup Report</title>
</head>
<body style="margin:0;padding:0;background:#0d1117;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#0d1117;padding:40px 16px;">
    <tr>
      <td align="center">
        <table width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%%;">

          <!-- Header -->
          <tr>
            <td style="background:#161b22;border-radius:12px 12px 0 0;padding:28px 32px;border:1px solid #30363d;border-bottom:none;">
              <table width="100%%" cellpadding="0" cellspacing="0">
                <tr>
                  <td>
                    <div style="font-size:22px;font-weight:700;color:#f0f6fc;line-height:1;">🐙 GitHub Backup</div>
                    <div style="margin-top:12px;">
                      <span style="display:inline-block;padding:5px 14px;border-radius:20px;font-size:12px;font-weight:600;background:%s;color:%s;">%s</span>
                    </div>
                  </td>
                  <td align="right" style="vertical-align:top;">
                    <div style="color:#8b949e;font-size:12px;white-space:nowrap;">%s</div>
                    <div style="color:#8b949e;font-size:12px;margin-top:5px;white-space:nowrap;">⏱ %s</div>
                  </td>
                </tr>
              </table>
            </td>
          </tr>

          <!-- Stats Cards -->
          <tr>
            <td style="background:#161b22;padding:20px 32px 8px;border-left:1px solid #30363d;border-right:1px solid #30363d;">
              <table width="100%%" cellpadding="0" cellspacing="0">
                <tr>
                  <td width="33%%" style="padding:0 6px 0 0;">
                    <div style="background:#21262d;border-radius:8px;padding:18px;text-align:center;border:1px solid #30363d;">
                      <div style="font-size:30px;font-weight:700;color:#f0f6fc;line-height:1;">%d</div>
                      <div style="font-size:11px;color:#8b949e;margin-top:6px;text-transform:uppercase;letter-spacing:0.8px;">Total</div>
                    </div>
                  </td>
                  <td width="33%%" style="padding:0 3px;">
                    <div style="background:#1a2e22;border-radius:8px;padding:18px;text-align:center;border:1px solid #2ea043;">
                      <div style="font-size:30px;font-weight:700;color:#3fb950;line-height:1;">%d</div>
                      <div style="font-size:11px;color:#8b949e;margin-top:6px;text-transform:uppercase;letter-spacing:0.8px;">Success</div>
                    </div>
                  </td>
                  <td width="33%%" style="padding:0 0 0 6px;">
                    <div style="background:%s;border-radius:8px;padding:18px;text-align:center;border:1px solid %s;">
                      <div style="font-size:30px;font-weight:700;color:%s;line-height:1;">%d</div>
                      <div style="font-size:11px;color:#8b949e;margin-top:6px;text-transform:uppercase;letter-spacing:0.8px;">Failed</div>
                    </div>
                  </td>
                </tr>
              </table>
            </td>
          </tr>

          <!-- Progress Bar -->
          <tr>
            <td style="background:#161b22;padding:16px 32px 20px;border-left:1px solid #30363d;border-right:1px solid #30363d;">
              <div style="height:6px;background:#21262d;border-radius:3px;overflow:hidden;">
                <div style="height:100%%;width:%d%%;background:linear-gradient(90deg,#3fb950,#2ea043);border-radius:3px;"></div>
              </div>
              <div style="margin-top:6px;font-size:11px;color:#8b949e;text-align:right;">%d%% success rate</div>
            </td>
          </tr>

          <!-- Backup Location -->
          <tr>
            <td style="background:#161b22;padding:0 32px 24px;border-left:1px solid #30363d;border-right:1px solid #30363d;">
              <div style="background:#0d1117;border-radius:6px;padding:12px 16px;border:1px solid #30363d;">
                <span style="color:#8b949e;font-size:12px;margin-right:8px;">📁 Backup Path</span>
                <span style="color:#79c0ff;font-size:13px;font-family:'SFMono-Regular',Consolas,'Liberation Mono',Menlo,monospace;">%s</span>
              </div>
            </td>
          </tr>

          %s

          <!-- Footer -->
          <tr>
            <td style="background:#0d1117;border-radius:0 0 12px 12px;padding:18px 32px;border:1px solid #30363d;border-top-color:#21262d;">
              <table width="100%%" cellpadding="0" cellspacing="0">
                <tr>
                  <td style="color:#484f58;font-size:11px;padding-bottom:8px;" colspan="2">
                    <span style="color:#8b949e;font-size:11px;font-weight:600;">🖥 Machine</span>
                    <span style="display:inline-block;margin-left:10px;background:#21262d;border:1px solid #30363d;border-radius:4px;padding:2px 8px;font-family:'SFMono-Regular',Consolas,monospace;font-size:11px;color:#79c0ff;">%s</span>
                    <span style="display:inline-block;margin-left:6px;background:#21262d;border:1px solid #30363d;border-radius:4px;padding:2px 8px;font-family:'SFMono-Regular',Consolas,monospace;font-size:11px;color:#d2a8ff;">%s / %s</span>
                  </td>
                </tr>
                <tr>
                  <td style="color:#484f58;font-size:11px;">Automated by <span style="color:#8b949e;">github-backup</span></td>
                  <td align="right" style="color:#484f58;font-size:11px;">© %s</td>
                </tr>
              </table>
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		badgeBg, badgeColor, badgeText,
		runDate, dur,
		d.Total, d.Success,
		failCardBg, failCardBorder, failCardColor, d.FailCount,
		successPct, successPct,
		html.EscapeString(d.Location),
		failedSection,
		html.EscapeString(hostname), html.EscapeString(osName), html.EscapeString(arch),
		year,
	)
}
