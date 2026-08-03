package services

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"backend/internal/logger"
	"backend/internal/metrics"

	"go.uber.org/zap"
)

type VSIService struct{}

var VSI = &VSIService{}

func (v *VSIService) Connect(sandboxID uint) (time.Duration, error) {
	start := time.Now()

	// Check if global failure config overrides this with VSI timeout simulation
	FailureConfig.Lock()
	vsiTimeoutSimulated := FailureConfig.VSITimeout
	FailureConfig.Unlock()

	logger.Log.Info("Starting VSI connection attempt", zap.Uint("sandbox_id", sandboxID))

	if vsiTimeoutSimulated {
		// Simulate full timeout (sleep 6 seconds and return timeout error)
		time.Sleep(6 * time.Second)
		duration := time.Since(start)

		metrics.VsiConnectionTotal.WithLabelValues("timeout").Inc()
		metrics.VsiConnectionFailedTotal.WithLabelValues("timeout_override").Inc()
		metrics.VsiConnectionDuration.Observe(duration.Seconds())

		logger.Log.Error("VSI connection failed: Timeout override simulation active",
			zap.Uint("sandbox_id", sandboxID),
			zap.Duration("duration", duration),
		)
		return duration, errors.New("connection timed out (simulated)")
	}

	// Normal infrastructure simulation
	// 70% Success, 20% Delay (5-10s), 10% Failure
	roll := rand.Intn(100)
	var err error
	var duration time.Duration

	if roll < 70 {
		// Success: 100ms - 500ms latency
		latencyMs := 100 + rand.Intn(400)
		duration = time.Duration(latencyMs) * time.Millisecond
		time.Sleep(duration)

		metrics.VsiConnectionTotal.WithLabelValues("success").Inc()
		logger.Log.Info("VSI connection succeeded",
			zap.Uint("sandbox_id", sandboxID),
			zap.Duration("duration", duration),
		)
	} else if roll < 90 {
		// Delay: 5 - 10s
		delaySec := 5 + rand.Intn(6)
		duration = time.Duration(delaySec) * time.Second
		time.Sleep(duration)

		metrics.VsiConnectionTotal.WithLabelValues("success_delayed").Inc()
		logger.Log.Info("VSI connection succeeded after infrastructure delay",
			zap.Uint("sandbox_id", sandboxID),
			zap.Duration("duration", duration),
		)
	} else {
		// Failure: 500ms delay then fail
		duration = 500 * time.Millisecond
		time.Sleep(duration)
		err = errors.New("simulated remote host connection refused (500)")

		metrics.VsiConnectionTotal.WithLabelValues("failure").Inc()
		metrics.VsiConnectionFailedTotal.WithLabelValues("connection_refused").Inc()
		logger.Log.Error("VSI connection failed: Remote host connection refused",
			zap.Uint("sandbox_id", sandboxID),
			zap.Duration("duration", duration),
		)
	}

	metrics.VsiConnectionDuration.Observe(duration.Seconds())
	return duration, err
}
func (v *VSIService) RunCommand(sandboxID uint, command string) (string, error) {
	logger.Log.Info("Executing remote command on VSI", zap.Uint("sandbox_id", sandboxID), zap.String("command", command))

	// Simulate command execution delay
	time.Sleep(200 * time.Millisecond)

	// Simple check for simulated error commands
	if command == "fail" || command == "sudo rm -rf /" {
		logger.Log.Error("VSI command execution failed", zap.Uint("sandbox_id", sandboxID), zap.String("command", command))
		return "", fmt.Errorf("bash: command execution failed for: '%s'", command)
	}

	var output string
	switch command {
	case "uname -a":
		output = fmt.Sprintf("Linux Sandbox-VSI-%d 5.15.0-88-generic #98-Ubuntu SMP Mon Oct 2 15:18:56 UTC 2023 x86_64 x86_64 x86_64 GNU/Linux", sandboxID)
	case "df -h":
		output = "Filesystem      Size  Used Avail Use% Mounted on\n" +
			"/dev/sda1        40G   14G   26G  35% /\n" +
			"tmpfs           3.9G     0  3.9G   0% /dev/shm\n" +
			"/dev/sdb1       100G   24G   76G  24% /data"
	case "free -m", "free -h":
		output = "              total        used        free      shared  buff/cache   available\n" +
			"Mem:          7.8Gi       2.4Gi       3.1Gi       120Mi       2.3Gi       5.1Gi\n" +
			"Swap:         2.0Gi          0B       2.0Gi"
	case "uptime":
		output = fmt.Sprintf(" %s up 3 days, 14:22,  1 user,  load average: 0.12, 0.08, 0.05", time.Now().Format("15:04:05"))
	default:
		output = fmt.Sprintf("Sandbox-VSI-%d: command '%s' completed successfully.", sandboxID, command)
	}

	logger.Log.Info("VSI command execution completed", zap.Uint("sandbox_id", sandboxID), zap.String("command", command))
	return output, nil
}
