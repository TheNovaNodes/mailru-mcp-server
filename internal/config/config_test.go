package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromEnv_Success(t *testing.T) {
	os.Setenv("MAILRU_USERNAME", "test@mail.ru")
	os.Setenv("MAILRU_APP_PASS", "secretpass")
	os.Setenv("MAILRU_TIMEOUT", "20")
	defer func() {
		os.Unsetenv("MAILRU_USERNAME")
		os.Unsetenv("MAILRU_APP_PASS")
		os.Unsetenv("MAILRU_TIMEOUT")
	}()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Username != "test@mail.ru" {
		t.Errorf("expected username test@mail.ru, got %s", cfg.Username)
	}
	if cfg.Password != "secretpass" {
		t.Errorf("expected password secretpass, got %s", cfg.Password)
	}
	if cfg.IMAPHost != "imap.mail.ru" {
		t.Errorf("expected IMAP host imap.mail.ru, got %s", cfg.IMAPHost)
	}
	if cfg.SMTPHost != "smtp.mail.ru" {
		t.Errorf("expected SMTP host smtp.mail.ru, got %s", cfg.SMTPHost)
	}
	if cfg.WebDAVHost != "https://webdav.cloud.mail.ru" {
		t.Errorf("expected WebDAV host https://webdav.cloud.mail.ru, got %s", cfg.WebDAVHost)
	}
	if cfg.Timeout != 20*time.Second {
		t.Errorf("expected timeout 20s, got %v", cfg.Timeout)
	}
}

func TestLoadFromEnv_MissingCredentials(t *testing.T) {
	os.Unsetenv("MAILRU_USERNAME")
	os.Unsetenv("MAILRU_APP_PASS")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error on missing credentials, got nil")
	}

	os.Setenv("MAILRU_USERNAME", "user@mail.ru")
	os.Unsetenv("MAILRU_APP_PASS")
	_, err = LoadFromEnv()
	if err == nil {
		t.Fatal("expected error on missing password, got nil")
	}
	os.Unsetenv("MAILRU_USERNAME")
}

func TestLoadFromEnv_CustomHostsAndDefaultTimeout(t *testing.T) {
	os.Setenv("MAILRU_USERNAME", "user@mail.ru")
	os.Setenv("MAILRU_APP_PASS", "pass")
	os.Setenv("MAILRU_IMAP_HOST", "custom.imap.mail.ru")
	os.Setenv("MAILRU_SMTP_HOST", "custom.smtp.mail.ru")
	os.Setenv("MAILRU_WEBDAV_HOST", "https://custom.webdav.mail.ru")
	os.Setenv("MAILRU_TIMEOUT", "invalid")
	defer func() {
		os.Unsetenv("MAILRU_USERNAME")
		os.Unsetenv("MAILRU_APP_PASS")
		os.Unsetenv("MAILRU_IMAP_HOST")
		os.Unsetenv("MAILRU_SMTP_HOST")
		os.Unsetenv("MAILRU_WEBDAV_HOST")
		os.Unsetenv("MAILRU_TIMEOUT")
	}()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.IMAPHost != "custom.imap.mail.ru" {
		t.Errorf("got %s", cfg.IMAPHost)
	}
	if cfg.SMTPHost != "custom.smtp.mail.ru" {
		t.Errorf("got %s", cfg.SMTPHost)
	}
	if cfg.WebDAVHost != "https://custom.webdav.mail.ru" {
		t.Errorf("got %s", cfg.WebDAVHost)
	}
	if cfg.Timeout != 15*time.Second {
		t.Errorf("expected fallback 15s timeout, got %v", cfg.Timeout)
	}
}
