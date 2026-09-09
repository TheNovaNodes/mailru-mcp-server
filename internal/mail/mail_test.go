package mail

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	gomail "github.com/emersion/go-message/mail"
)

func TestMail_Constructor(t *testing.T) {
	_, err := NewLiveClient("", "pass", "", "", 0)
	if err == nil {
		t.Fatal("expected error on empty username")
	}

	cli, err := NewLiveClient("user@mail.ru", "pass", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cli.IMAPHost != "imap.mail.ru" {
		t.Errorf("expected default imap host, got %s", cli.IMAPHost)
	}
	if cli.SMTPHost != "smtp.mail.ru" {
		t.Errorf("expected default smtp host, got %s", cli.SMTPHost)
	}
	if cli.Timeout != 15*time.Second {
		t.Errorf("expected default timeout 15s, got %v", cli.Timeout)
	}
}

func TestMail_FormatAddresses(t *testing.T) {
	addrs := []imap.Address{
		{Name: "Ivan Ivanov", Mailbox: "ivan", Host: "mail.ru"},
		{Name: "", Mailbox: "support", Host: "mail.ru"},
	}

	formatted := FormatAddresses(addrs)
	expected := "Ivan Ivanov <ivan@mail.ru>, support@mail.ru"
	if formatted != expected {
		t.Errorf("expected '%s', got '%s'", expected, formatted)
	}
}

func TestMail_BuildMIMEMessage(t *testing.T) {
	tmpDir := t.TempDir()
	attFile := filepath.Join(tmpDir, "report.pdf")
	_ = os.WriteFile(attFile, []byte("PDF-DATA-12345"), 0644)

	msgBytes := BuildMIMEMessage("me@mail.ru", "boss@mail.ru", "Monthly Report", "Here is your report.", attFile)
	msgStr := string(msgBytes)

	if !strings.Contains(msgStr, "From: me@mail.ru") {
		t.Error("missing From header")
	}
	if !strings.Contains(msgStr, "To: boss@mail.ru") {
		t.Error("missing To header")
	}
	if !strings.Contains(msgStr, "multipart/mixed") {
		t.Error("expected multipart/mixed for attachment")
	}
	if !strings.Contains(msgStr, "report.pdf") {
		t.Error("expected attachment filename in body")
	}

	// Message without attachment
	plainBytes := BuildMIMEMessage("me@mail.ru", "boss@mail.ru", "Hello", "Simple text", "")
	plainStr := string(plainBytes)
	if strings.Contains(plainStr, "multipart/mixed") {
		t.Error("did not expect multipart for plain email")
	}
	if !strings.Contains(plainStr, "Simple text") {
		t.Error("missing body text")
	}
}

func TestMail_ExtractTextAndHTML(t *testing.T) {
	// Empty payload
	t1, h1 := ExtractTextAndHTML(nil)
	if t1 != "" || h1 != "" {
		t.Errorf("expected empty, got %s, %s", t1, h1)
	}

	// Plain raw bytes fallback
	raw := []byte("Direct raw message string")
	t2, _ := ExtractTextAndHTML(raw)
	if !strings.Contains(t2, "Direct raw message string") {
		t.Errorf("got %s", t2)
	}

	// Structured MIME message with text and html parts
	var b bytes.Buffer
	var h gomail.Header
	h.SetAddressList("From", []*gomail.Address{{Name: "Sender", Address: "s@mail.ru"}})
	h.SetSubject("MIME Test")
	mw, err := gomail.CreateWriter(&b, h)
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}

	// Inline writer
	iw, err := mw.CreateInline()
	if err == nil {
		var textHeader gomail.InlineHeader
		textHeader.Set("Content-Type", "text/plain; charset=utf-8")
		tw, err := iw.CreatePart(textHeader)
		if err == nil {
			_, _ = tw.Write([]byte("Plain text version"))
			_ = tw.Close()
		}

		var htmlHeader gomail.InlineHeader
		htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
		hw, err := iw.CreatePart(htmlHeader)
		if err == nil {
			_, _ = hw.Write([]byte("<h1>HTML version</h1>"))
			_ = hw.Close()
		}
		_ = iw.Close()
	}

	_ = mw.Close()

	t3, h3 := ExtractTextAndHTML(b.Bytes())
	if !strings.Contains(t3, "Plain text version") {
		t.Errorf("expected plain text, got %s", t3)
	}
	if !strings.Contains(h3, "<h1>HTML version</h1>") {
		t.Errorf("expected HTML content, got %s", h3)
	}
}

func TestMail_MockClient(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	mock.Emails = []EmailSummary{
		{UID: "101", Folder: "INBOX", Subject: "Welcome", From: "admin@mail.ru"},
		{UID: "102", Folder: "INBOX", Subject: "Invoice", From: "billing@mail.ru"},
	}
	mock.SearchItems = []EmailSearchItem{
		{UID: "102", Folder: "INBOX", Subject: "Invoice", TextSnippet: "Payment due for services"},
	}
	mock.Details["102"] = &EmailDetails{
		UID:     "102",
		Folder:  "INBOX",
		Subject: "Invoice",
		Text:    "Payment details inside",
	}

	// FetchRecentEmails
	emails, err := mock.FetchRecentEmails(ctx, 1, "INBOX", true)
	if err != nil || len(emails) != 1 || emails[0].UID != "101" {
		t.Fatalf("FetchRecentEmails failed: %v, len=%d", err, len(emails))
	}

	// SearchEmails
	results, err := mock.SearchEmails(ctx, "Invoice", "INBOX")
	if err != nil || len(results) != 1 {
		t.Fatalf("SearchEmails failed: %v", err)
	}

	// GetEmailBody
	body, err := mock.GetEmailBody(ctx, "102", "INBOX")
	if err != nil || body.Text != "Payment details inside" {
		t.Fatalf("GetEmailBody failed: %v", err)
	}

	// GetEmailBody not found
	_, err = mock.GetEmailBody(ctx, "999", "INBOX")
	if err == nil {
		t.Fatal("expected not found error")
	}

	// MoveMessage
	if err := mock.MoveMessage(ctx, "102", "Archive", "INBOX"); err != nil {
		t.Fatalf("MoveMessage failed: %v", err)
	}
	if mock.MovedMessages["102"] != "Archive" {
		t.Errorf("expected moved to Archive, got %s", mock.MovedMessages["102"])
	}

	// SaveDraft
	if err := mock.SaveDraft(ctx, "client@mail.ru", "Proposal", "Here is our offer"); err != nil {
		t.Fatalf("SaveDraft failed: %v", err)
	}
	if len(mock.SavedDrafts) != 1 {
		t.Errorf("expected 1 saved draft, got %d", len(mock.SavedDrafts))
	}

	// SendEmail
	if err := mock.SendEmail(ctx, "client@mail.ru", "Reply", "Thank you", ""); err != nil {
		t.Fatalf("SendEmail failed: %v", err)
	}
	if len(mock.SentEmails) != 1 {
		t.Errorf("expected 1 sent email, got %d", len(mock.SentEmails))
	}

	// Failures
	mock.ShouldFail = true
	if _, err := mock.FetchRecentEmails(ctx, 10, "INBOX", false); err == nil {
		t.Error("expected mock failure")
	}
	if _, err := mock.SearchEmails(ctx, "query", "INBOX"); err == nil {
		t.Error("expected mock failure")
	}
	if _, err := mock.GetEmailBody(ctx, "102", "INBOX"); err == nil {
		t.Error("expected mock failure")
	}
	if err := mock.MoveMessage(ctx, "102", "A", "B"); err == nil {
		t.Error("expected mock failure")
	}
	if err := mock.SaveDraft(ctx, "a", "b", "c"); err == nil {
		t.Error("expected mock failure")
	}
	if err := mock.SendEmail(ctx, "a", "b", "c", ""); err == nil {
		t.Error("expected mock failure")
	}
}

func TestMail_LiveClient_ClosedPortErrors(t *testing.T) {
	cli, err := NewLiveClient("test@mail.ru", "pass", "127.0.0.1:65432", "127.0.0.1:65433", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	ctx := context.Background()

	// FetchRecentEmails error
	if _, err := cli.FetchRecentEmails(ctx, 5, "INBOX", false); err == nil {
		t.Error("expected error dialing closed IMAP port")
	}

	// SearchEmails error
	if _, err := cli.SearchEmails(ctx, "test", "INBOX"); err == nil {
		t.Error("expected error dialing closed IMAP port")
	}

	// GetEmailBody error
	if _, err := cli.GetEmailBody(ctx, "123", "INBOX"); err == nil {
		t.Error("expected error dialing closed IMAP port")
	}

	// MoveMessage error
	if err := cli.MoveMessage(ctx, "123", "Archive", "INBOX"); err == nil {
		t.Error("expected error dialing closed IMAP port")
	}

	// SaveDraft error
	if err := cli.SaveDraft(ctx, "a@mail.ru", "sub", "body"); err == nil {
		t.Error("expected error dialing closed IMAP port")
	}

	// SendEmail error
	if err := cli.SendEmail(ctx, "a@mail.ru", "sub", "body", ""); err == nil {
		t.Error("expected error dialing closed SMTP port")
	}
}

