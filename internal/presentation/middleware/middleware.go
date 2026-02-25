package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Printf("[%d] %s %s %v", status, c.Request.Method, path, latency)
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func BasicAuth(username, password string) gin.HandlerFunc {
	users := map[string]string{
		username: password,
	}

	return func(c *gin.Context) {
		u, p, ok := c.Request.BasicAuth()
		if !ok {
			c.AbortWithStatus(401)
			return
		}

		if expectedPass, exists := users[u]; !exists || expectedPass != p {
			c.AbortWithStatus(401)
			return
		}

		c.Next()
	}
}
