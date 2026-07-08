package middleware

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"backend/internal/api/services"
	"backend/internal/logger"
	"backend/internal/metrics"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Generate a unique Request ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	_, _ = cryptorand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// RequestIDMiddleware injects a unique X-Request-ID into headers and context
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}
		c.Set("request_id", reqID)
		c.Header("X-Request-ID", reqID)
		c.Next()
	}
}

// CORSMiddleware manages Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "POST, HEAD, PATCH, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// RecoveryMiddleware catches panics, logs stacktrace, and increments metrics
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				reqID := c.GetString("request_id")
				method := c.Request.Method
				path := c.Request.URL.Path

				logger.Log.Error("PANIC RECOVERED",
					zap.Any("error", err),
					zap.String("request_id", reqID),
					zap.String("method", method),
					zap.String("path", path),
					zap.String("stack", string(debug.Stack())),
				)

				// Increment Prometheus error counts
				metrics.HttpRequestsTotal.WithLabelValues(method, path, "500").Inc()
				metrics.HttpRequestErrorsTotal.WithLabelValues(method, path, "500").Inc()

				c.JSON(http.StatusInternalServerError, gin.H{
					"error":      "Internal Server Error",
					"request_id": reqID,
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// LoggingMiddleware logs detailed structured JSON requests
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// Read response body length or details if needed
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		reqID := c.GetString("request_id")
		userAgent := c.Request.UserAgent()
		clientIP := c.ClientIP()

		if rawQuery != "" {
			path = path + "?" + rawQuery
		}

		errMsg := ""
		if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}

		logFields := []zap.Field{
			zap.String("request_id", reqID),
			zap.String("method", c.Request.Method),
			zap.String("endpoint", c.Request.URL.Path),
			zap.String("path", path),
			zap.Int("status_code", status),
			zap.Duration("latency", latency),
			zap.String("latency_human", latency.String()),
			zap.String("user_agent", userAgent),
			zap.String("remote_ip", clientIP),
			zap.String("log_level", "INFO"),
		}

		if errMsg != "" {
			logFields = append(logFields, zap.String("error_msg", errMsg))
		}

		// Log warning/error level depending on HTTP code
		if status >= 500 {
			logFields[len(logFields)-1] = zap.String("log_level", "ERROR")
			logger.Log.Error("HTTP Request Failed", logFields...)
		} else if status >= 400 {
			logFields[len(logFields)-1] = zap.String("log_level", "WARNING")
			logger.Log.Warn("HTTP Request Client Warning", logFields...)
		} else {
			logger.Log.Info("HTTP Request Completed", logFields...)
		}
	}
}

// PrometheusMiddleware monitors traffic volume, errors, latency and concurrency
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		// FullPath retrieves standard route registration format like /sandbox/:id instead of actual ID
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unknown"
		}

		metrics.HttpRequestsInProgress.WithLabelValues(method, endpoint).Inc()
		defer metrics.HttpRequestsInProgress.WithLabelValues(method, endpoint).Dec()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		metrics.HttpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		metrics.HttpRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration)

		if c.Writer.Status() >= 400 {
			metrics.HttpRequestErrorsTotal.WithLabelValues(method, endpoint, status).Inc()
		}
	}
}

// TokenBucket Rate Limiter per Client IP
type ipLimiter struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	sync.Mutex
	limiters   map[string]*ipLimiter
	rate       float64 // tokens per second
	capacity   float64 // max bucket size
}

func NewRateLimiter(rate float64, capacity float64) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*ipLimiter),
		rate:     rate,
		capacity: capacity,
	}
}

func (rl *RateLimiter) Limit(ip string) bool {
	rl.Lock()
	defer rl.Unlock()

	limiter, exists := rl.limiters[ip]
	now := time.Now()

	if !exists {
		rl.limiters[ip] = &ipLimiter{
			tokens:     rl.capacity,
			lastRefill: now,
		}
		return true
	}

	elapsed := now.Sub(limiter.lastRefill).Seconds()
	limiter.tokens = limiter.tokens + (elapsed * rl.rate)
	if limiter.tokens > rl.capacity {
		limiter.tokens = rl.capacity
	}
	limiter.lastRefill = now

	if limiter.tokens >= 1.0 {
		limiter.tokens -= 1.0
		return true
	}

	return false
}

// RateLimitMiddleware blocks IPs exceeding rate guidelines (e.g. max 50 req/sec, capacity 100)
func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.Limit(ip) {
			reqID := c.GetString("request_id")
			logger.Log.Warn("Rate limit exceeded", zap.String("ip", ip), zap.String("request_id", reqID))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":      "Too Many Requests - Rate Limit Exceeded",
				"request_id": reqID,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// TimeoutMiddleware interrupts long-running requests exceeding limit
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// Channel to notify completion
		done := make(chan struct{})
		
		// Create a custom response writer to buffer output
		writer := &timeoutWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// Request completed on time. Write buffered body if headers weren't sent.
			if !writer.HeaderWritten() {
				_, _ = writer.ResponseWriter.Write(writer.body.Bytes())
			}
		case <-ctx.Done():
			// Timeout occurred
			reqID := c.GetString("request_id")
			logger.Log.Error("Request Timeout Triggered", zap.String("request_id", reqID), zap.Duration("limit", timeout))
			
			c.Writer.Header().Set("Content-Type", "application/json")
			c.Writer.WriteHeader(http.StatusGatewayTimeout)
			_, _ = c.Writer.Write([]byte(fmt.Sprintf(`{"error":"Request Gateway Timeout","request_id":"%s"}`, reqID)))
			c.Abort()
		}
	}
}

type timeoutWriter struct {
	gin.ResponseWriter
	body           *bytes.Buffer
	headerWritten  bool
}

func (tw *timeoutWriter) WriteHeader(statusCode int) {
	tw.headerWritten = true
	tw.ResponseWriter.WriteHeader(statusCode)
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	if tw.headerWritten {
		return tw.ResponseWriter.Write(b)
	}
	return tw.body.Write(b)
}

func (tw *timeoutWriter) HeaderWritten() bool {
	return tw.headerWritten
}

// FailureSimulatorMiddleware injects API delay and random 500 errors dynamically
func FailureSimulatorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		services.FailureConfig.Lock()
		apiDelay := services.FailureConfig.APIDelay
		randomErrors := services.FailureConfig.RandomErrors
		services.FailureConfig.Unlock()

		if apiDelay > 0 {
			time.Sleep(apiDelay)
		}

		if randomErrors {
			// Inject 25% random failures on normal endpoints (excluding health and metrics targets)
			path := c.Request.URL.Path
			if path != "/health" && path != "/metrics" && rand.Float32() < 0.25 {
				reqID := c.GetString("request_id")
				logger.Log.Error("Simulated random HTTP API failure triggered", zap.String("request_id", reqID))
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":      "Internal Server Error (Simulated random API failure)",
					"request_id": reqID,
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
