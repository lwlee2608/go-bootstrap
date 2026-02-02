package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			slog.Error("Request error",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"error", err.Error())

			if !c.Writer.Written() {
				c.JSON(500, gin.H{
					"error": "Internal server error",
				})
			}
		}
	}
}
