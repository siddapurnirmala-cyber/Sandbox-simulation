package services

import (
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"sync"
	"time"

	"backend/internal/logger"
	"go.uber.org/zap"
)

type EmailService struct {
	sync.Mutex
	lastAlertTime map[string]time.Time
}

var Email = &EmailService{
	lastAlertTime: make(map[string]time.Time),
}

func (s *EmailService) SendLatencyAlert(method, path, requestID string, latency time.Duration, clientIP string, reason string, action string) {
	// 2-minute cooldown rate-limiting per alert source/path
	const alertCooldown = 2 * time.Minute
	s.Lock()
	lastTime, exists := s.lastAlertTime[path]
	shouldAlert := !exists || time.Since(lastTime) >= alertCooldown
	if shouldAlert {
		s.lastAlertTime[path] = time.Now()
	}
	s.Unlock()

	if !shouldAlert {
		logger.Log.Info("SLA breach detected but alert skipped due to 2-minute alert cooldown",
			zap.String("path", path),
			zap.Duration("latency", latency),
		)
		return
	}

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	alertReceiver := os.Getenv("ALERT_RECEIVER")

	// If SMTP is not configured, skip sending but log a warning
	if smtpHost == "" || smtpUser == "" || smtpPassword == "" || alertReceiver == "" {
		logger.Log.Warn("Email alerts are enabled but SMTP credentials are not fully configured in docker-compose.yml",
			zap.String("smtp_host", smtpHost),
			zap.String("smtp_user", smtpUser),
			zap.String("alert_receiver", alertReceiver),
		)
		return
	}

	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil || smtpPort == 0 {
		smtpPort = 587 // default TLS port
	}

	subject := fmt.Sprintf("[ALERT] %s on %s", reason, path)
	
	// Create styled HTML email body
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: 'Segoe UI', Arial, sans-serif; color: #333; line-height: 1.6; }
        .card { max-width: 600px; margin: 20px auto; border: 1px solid #e1e8ed; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 6px rgba(0,0,0,0.05); }
        .header { background: #e11d48; color: white; padding: 20px; text-align: center; }
        .header h1 { margin: 0; font-size: 20px; }
        .content { padding: 30px; background: #fff; }
        .metric-table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
        .metric-table td { padding: 12px; border-bottom: 1px solid #f0f0f0; }
        .metric-table td.label { font-weight: bold; color: #64748b; width: 150px; }
        .metric-table td.value { font-family: monospace; font-size: 14px; color: #0f172a; }
        .footer { background: #f8fafc; padding: 20px; text-align: center; font-size: 12px; color: #94a3b8; border-top: 1px solid #e2e8f0; }
        .badge { background: #fee2e2; color: #ef4444; padding: 4px 8px; border-radius: 4px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="card">
        <div class="header">
            <h1>SLA Threshold Breach Alert</h1>
        </div>
        <div class="content">
            <p>A performance SLA breach occurred on the <strong>Sandbox Observability Platform</strong>.</p>
            <table class="metric-table">
                <tr>
                    <td class="label">Breach Reason</td>
                    <td class="value" style="font-weight: bold; color: #e11d48;">%s</td>
                </tr>
                <tr>
                    <td class="label">Triggered At</td>
                    <td class="value">%s</td>
                </tr>
                <tr>
                    <td class="label">Method</td>
                    <td class="value">%s</td>
                </tr>
                <tr>
                    <td class="label">Request Path</td>
                    <td class="value">%s</td>
                </tr>
                <tr>
                    <td class="label">Request ID</td>
                    <td class="value">%s</td>
                </tr>
                <tr>
                    <td class="label">Latency / Load Time</td>
                    <td class="value"><span class="badge">%s</span></td>
                </tr>
                <tr>
                    <td class="label">Client IP</td>
                    <td class="value">%s</td>
                </tr>
            </table>
            <p><strong>Recommended Action:</strong> %s</p>
        </div>
        <div class="footer">
            This is an automated alert sent by the Sandbox Observability Platform.
        </div>
    </div>
</body>
</html>`,
		reason,
		time.Now().Format(time.RFC1123),
		method,
		path,
		requestID,
		latency.String(),
		clientIP,
		action,
	)

	// Format SMTP Message headers
	msg := []byte(
		"From: " + smtpUser + "\r\n" +
		"To: " + alertReceiver + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n",
	)

	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)

	// Send using standard net/smtp
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	err = smtp.SendMail(addr, auth, smtpUser, []string{alertReceiver}, msg)
	if err != nil {
		logger.Log.Error("Failed to send latency alert email over SMTP",
			zap.Error(err),
			zap.String("smtp_host", smtpHost),
			zap.String("smtp_port", smtpPortStr),
			zap.String("smtp_user", smtpUser),
		)
		return
	}

	logger.Log.Info("Latency SLA breach alert email sent successfully",
		zap.String("recipient", alertReceiver),
		zap.String("latency", latency.String()),
	)
}
