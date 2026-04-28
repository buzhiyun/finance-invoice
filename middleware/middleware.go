package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/buzhiyun/finance-invoice/auth"
	"github.com/gin-gonic/gin"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		claims, err := auth.ValidateToken(tokenStr, jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("username", claims.Username)
		c.Next()
	}
}

func IPWhitelist(allowedNets []*net.IPNet) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(allowedNets) == 0 {
			c.Next()
			return
		}

		clientIP := extractClientIP(c)
		ip := net.ParseIP(clientIP)
		if ip == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid client IP"})
			return
		}

		for _, ipNet := range allowedNets {
			if ipNet.Contains(ip) {
				c.Next()
				return
			}
		}

		log.Printf("[IP白名单] 拦截访问: IP=%s, 方法=%s, 路径=%s", clientIP, c.Request.Method, c.Request.URL.Path)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "IP not allowed"})
	}
}

func extractClientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	return ip
}
