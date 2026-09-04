package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// DefaultTimeout is the standard timeout in seconds for all network I/O.
const DefaultTimeout = 15

// Config holds all runtime configurations for the Mail.ru MCP Server.
type Config struct {
	Username   string
	Password   string
	IMAPHost   string
	SMTPHost   string
	WebDAVHost string
	Timeout    time.Duration
}

// LoadFromEnv loads credentials and endpoint configurations from environment variables.
func LoadFromEnv() (*Config, error) {
	username := os.Getenv("MAILRU_USERNAME")
	password := os.Getenv("MAILRU_APP_PASS")
	if username == "" || password == "" {
		return nil, errors.New("MAILRU_USERNAME and MAILRU_APP_PASS must be set in environment")
	}

	imapHost := os.Getenv("MAILRU_IMAP_HOST")
	if imapHost == "" {
		imapHost = "imap.mail.ru"
	}

	smtpHost := os.Getenv("MAILRU_SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "smtp.mail.ru"
	}

	webdavHost := os.Getenv("MAILRU_WEBDAV_HOST")
	if webdavHost == "" {
		webdavHost = "https://webdav.cloud.mail.ru"
	}

	timeoutSec := DefaultTimeout
	if tStr := os.Getenv("MAILRU_TIMEOUT"); tStr != "" {
		if t, err := strconv.Atoi(tStr); err == nil && t > 0 {
			timeoutSec = t
		}
	}

	return &Config{
		Username:   username,
		Password:   password,
		IMAPHost:   imapHost,
		SMTPHost:   smtpHost,
		WebDAVHost: webdavHost,
		Timeout:    time.Duration(timeoutSec) * time.Second,
	}, nil
}
