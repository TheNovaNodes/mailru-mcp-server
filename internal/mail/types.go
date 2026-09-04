package mail

import (
	"context"
	"time"
)

// EmailSummary contains overview fields for an email.
type EmailSummary struct {
	UID     string    `json:"uid"`
	Folder  string    `json:"folder,omitempty"`
	Subject string    `json:"subject"`
	From    string    `json:"from"`
	To      string    `json:"to,omitempty"`
	Date    string    `json:"date"`
	DateRaw time.Time `json:"-"`
	Flags   []string  `json:"flags"`
	Text    string    `json:"text"`
}

// EmailSearchItem represents a search match.
type EmailSearchItem struct {
	UID         string `json:"uid"`
	Folder      string `json:"folder"`
	Subject     string `json:"subject"`
	From        string `json:"from"`
	Date        string `json:"date"`
	TextSnippet string `json:"text_snippet"`
}

// EmailDetails contains full content of a message.
type EmailDetails struct {
	UID     string `json:"uid"`
	Folder  string `json:"folder"`
	Subject string `json:"subject"`
	From    string `json:"from"`
	To      string `json:"to"`
	Date    string `json:"date"`
	Text    string `json:"text"`
	HTML    string `json:"html"`
}

// Client defines the interface for interacting with Mail.ru email.
type Client interface {
	FetchRecentEmails(ctx context.Context, limit int, folder string, includeSmartFolders bool) ([]EmailSummary, error)
	SearchEmails(ctx context.Context, query string, folder string) ([]EmailSearchItem, error)
	GetEmailBody(ctx context.Context, uid string, folder string) (*EmailDetails, error)
	MoveMessage(ctx context.Context, uid string, toFolder string, fromFolder string) error
	SaveDraft(ctx context.Context, toEmail, subject, body string) error
	SendEmail(ctx context.Context, toEmail, subject, body, attachmentPath string) error
}
