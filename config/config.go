package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ZhipuAPIKey   string
	UsersCSV      string
	AllowedIPs    []*net.IPNet
	MaxConcurrent int
	Port          string
	JWTSecret     string
}

func Load() (*Config, error) {
	cfg := &Config{
		ZhipuAPIKey:   os.Getenv("ZHIPU_API_KEY"),
		UsersCSV:      getEnvOrDefault("USERS_CSV", "users.csv"),
		MaxConcurrent: getEnvInt("MAX_CONCURRENT", 20),
		Port:          getEnvOrDefault("PORT", "8080"),
		JWTSecret:     getEnvOrDefault("JWT_SECRET", "finance-invoice-secret-key"),
	}

	if cfg.ZhipuAPIKey == "" {
		return nil, fmt.Errorf("ZHIPU_API_KEY is required")
	}

	allowedIPs := os.Getenv("ALLOWED_IPS")
	if allowedIPs != "" {
		for _, cidr := range strings.Split(allowedIPs, ",") {
			cidr = strings.TrimSpace(cidr)
			if !strings.Contains(cidr, "/") {
				cidr += "/32"
			}
			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
			}
			cfg.AllowedIPs = append(cfg.AllowedIPs, ipNet)
		}
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
