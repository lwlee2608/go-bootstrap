package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs all incoming HTTP requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		slog.Debug("Incoming request",
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"client_ip", c.ClientIP(),
		)

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		logLevel := slog.LevelDebug
		if statusCode >= 500 {
			logLevel = slog.LevelError
		} else if statusCode >= 400 {
			logLevel = slog.LevelWarn
		}

		slog.Log(c.Request.Context(), logLevel, "Request completed",
			"method", c.Request.Method,
			"path", path,
			"status", statusCode,
			"latency", latency.String(),
			"client_ip", c.ClientIP(),
		)
	}
}
