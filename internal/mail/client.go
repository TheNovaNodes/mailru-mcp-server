package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	gomail "github.com/emersion/go-message/mail"
)

// LiveClient connects to real IMAP and SMTP servers.
type LiveClient struct {
	Username string
	Password string
	IMAPHost string
	SMTPHost string
	Timeout  time.Duration
}

// NewLiveClient creates a new IMAP/SMTP client with strict timeouts.
func NewLiveClient(username, password, imapHost, smtpHost string, timeout time.Duration) (*LiveClient, error) {
	if username == "" || password == "" {
		return nil, errors.New("mail username and password must not be empty")
	}
	if imapHost == "" {
		imapHost = "imap.mail.ru"
	}
	if smtpHost == "" {
		smtpHost = "smtp.mail.ru"
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &LiveClient{
		Username: username,
		Password: password,
		IMAPHost: imapHost,
		SMTPHost: smtpHost,
		Timeout:  timeout,
	}, nil
}

func (c *LiveClient) dialIMAP() (*imapclient.Client, error) {
	dialer := &net.Dialer{Timeout: c.Timeout}
	port := "993"
	host := c.IMAPHost
	if strings.Contains(host, ":") {
		h, p, err := net.SplitHostPort(host)
		if err == nil {
			host = h
			port = p
		}
	}
	tlsConn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, port), &tls.Config{
		ServerName: host,
	})
	if err != nil {
		return nil, fmt.Errorf("IMAP TLS connection failed: %w", err)
	}

	options := &imapclient.Options{
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}
	client := imapclient.New(tlsConn, options)

	if err := client.Login(c.Username, c.Password).Wait(); err != nil {
		client.Close()
		return nil, fmt.Errorf("IMAP login failed: %w", err)
	}

	return client, nil
}

// FormatAddresses formats a slice of IMAP addresses into a string.
func FormatAddresses(addrs []imap.Address) string {
	var res []string
	for _, a := range addrs {
		if a.Name != "" {
			res = append(res, fmt.Sprintf("%s <%s@%s>", a.Name, a.Mailbox, a.Host))
		} else {
			res = append(res, fmt.Sprintf("%s@%s", a.Mailbox, a.Host))
		}
	}
	return strings.Join(res, ", ")
}

// ExtractTextAndHTML extracts text and html bodies from raw RFC822 bytes.
func ExtractTextAndHTML(raw []byte) (string, string) {
	if len(raw) == 0 {
		return "", ""
	}
	mr, err := gomail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return string(raw), ""
	}

	var textBody, htmlBody string
	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}
		switch h := p.Header.(type) {
		case *gomail.InlineHeader:
			contentType, _, _ := h.ContentType()
			b, _ := io.ReadAll(p.Body)
			if strings.HasPrefix(contentType, "text/plain") && textBody == "" {
				textBody = string(b)
			} else if strings.HasPrefix(contentType, "text/html") && htmlBody == "" {
				htmlBody = string(b)
			}
		}
	}

	if textBody == "" && htmlBody == "" {
		textBody = string(raw)
	}
	return textBody, htmlBody
}

func (c *LiveClient) FetchRecentEmails(ctx context.Context, limit int, folder string, includeSmartFolders bool) ([]EmailSummary, error) {
	client, err := c.dialIMAP()
	if err != nil {
		return nil, err
	}
	defer client.Logout()

	folders := []string{folder}
	if folder == "INBOX" && includeSmartFolders {
		folders = []string{"INBOX", "INBOX/Newsletters", "INBOX/Social", "INBOX/News", "INBOX/Receipts"}
	}

	var allSummaries []EmailSummary

	for _, fld := range folders {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		sel, err := client.Select(fld, &imap.SelectOptions{ReadOnly: true}).Wait()
		if err != nil || sel.NumMessages == 0 {
			continue
		}

		// Fetch messages in this folder
		fetchCount := uint32(limit)
		if fetchCount > sel.NumMessages {
			fetchCount = sel.NumMessages
		}
		fromSeq := sel.NumMessages - fetchCount + 1
		toSeq := sel.NumMessages

		var seqSet imap.SeqSet
		seqSet.AddRange(fromSeq, toSeq)

		fetchOpts := &imap.FetchOptions{
			Envelope:    true,
			UID:         true,
			Flags:       true,
			BodySection: []*imap.FetchItemBodySection{{}},
		}

		msgs, err := client.Fetch(seqSet, fetchOpts).Collect()
		if err != nil {
			continue
		}

		for _, msg := range msgs {
			var subj, fromStr, toStr string
			var dateVal time.Time
			if msg.Envelope != nil {
				subj = msg.Envelope.Subject
				fromStr = FormatAddresses(msg.Envelope.From)
				toStr = FormatAddresses(msg.Envelope.To)
				dateVal = msg.Envelope.Date.UTC()
			}
			if subj == "" {
				subj = "(No Subject)"
			}

			var flags []string
			for _, fl := range msg.Flags {
				flags = append(flags, string(fl))
			}

			var rawBody []byte
			for _, bs := range msg.BodySection {
				rawBody = bs.Bytes
				break
			}
			text, html := ExtractTextAndHTML(rawBody)
			body := text
			if body == "" {
				body = html
			}

			allSummaries = append(allSummaries, EmailSummary{
				UID:     strconv.FormatUint(uint64(msg.UID), 10),
				Folder:  fld,
				Subject: subj,
				From:    fromStr,
				To:      toStr,
				Date:    dateVal.Format(time.RFC3339),
				DateRaw: dateVal,
				Flags:   flags,
				Text:    body,
			})
		}
	}

	// Sort descending by date
	sort.Slice(allSummaries, func(i, j int) bool {
		return allSummaries[i].DateRaw.After(allSummaries[j].DateRaw)
	})

	if len(allSummaries) > limit {
		allSummaries = allSummaries[:limit]
	}

	return allSummaries, nil
}

func (c *LiveClient) SearchEmails(ctx context.Context, query string, folder string) ([]EmailSearchItem, error) {
	client, err := c.dialIMAP()
	if err != nil {
		return nil, err
	}
	defer client.Logout()

	_, err = client.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select folder %s: %w", folder, err)
	}

	criteria := &imap.SearchCriteria{
		Text: []string{query},
	}
	searchRes, err := client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var items []EmailSearchItem
	if len(searchRes.AllUIDs()) == 0 {
		return items, nil
	}

	uids := searchRes.AllUIDs()
	var uidSet imap.UIDSet
	for _, u := range uids {
		uidSet.AddNum(u)
	}

	fetchOpts := &imap.FetchOptions{
		Envelope:    true,
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{}},
	}

	msgs, err := client.Fetch(uidSet, fetchOpts).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch search results failed: %w", err)
	}

	for _, msg := range msgs {
		var subj, fromStr string
		var dateStr string
		if msg.Envelope != nil {
			subj = msg.Envelope.Subject
			fromStr = FormatAddresses(msg.Envelope.From)
			dateStr = msg.Envelope.Date.UTC().Format(time.RFC3339)
		}
		if subj == "" {
			subj = "(No Subject)"
		}

		var rawBody []byte
		for _, bs := range msg.BodySection {
			rawBody = bs.Bytes
			break
		}
		text, html := ExtractTextAndHTML(rawBody)
		snippet := text
		if snippet == "" {
			snippet = html
		}
		if len(snippet) > 500 {
			snippet = snippet[:500] + "..."
		}

		items = append(items, EmailSearchItem{
			UID:         strconv.FormatUint(uint64(msg.UID), 10),
			Folder:      folder,
			Subject:     subj,
			From:        fromStr,
			Date:        dateStr,
			TextSnippet: snippet,
		})
	}

	return items, nil
}

func (c *LiveClient) GetEmailBody(ctx context.Context, uid string, folder string) (*EmailDetails, error) {
	client, err := c.dialIMAP()
	if err != nil {
		return nil, err
	}
	defer client.Logout()

	_, err = client.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select folder %s: %w", folder, err)
	}

	parsedUID, err := strconv.ParseUint(uid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid UID format: %s", uid)
	}

	var uidSet imap.UIDSet
	uidSet.AddNum(imap.UID(parsedUID))

	fetchOpts := &imap.FetchOptions{
		Envelope:    true,
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{}},
	}

	msgs, err := client.Fetch(uidSet, fetchOpts).Collect()
	if err != nil || len(msgs) == 0 {
		return nil, fmt.Errorf("email UID %s not found in %s", uid, folder)
	}

	msg := msgs[0]
	var subj, fromStr, toStr, dateStr string
	if msg.Envelope != nil {
		subj = msg.Envelope.Subject
		fromStr = FormatAddresses(msg.Envelope.From)
		toStr = FormatAddresses(msg.Envelope.To)
		dateStr = msg.Envelope.Date.UTC().Format(time.RFC3339)
	}
	if subj == "" {
		subj = "(No Subject)"
	}

	var rawBody []byte
	for _, bs := range msg.BodySection {
		rawBody = bs.Bytes
		break
	}
	text, html := ExtractTextAndHTML(rawBody)

	return &EmailDetails{
		UID:     uid,
		Folder:  folder,
		Subject: subj,
		From:    fromStr,
		To:      toStr,
		Date:    dateStr,
		Text:    text,
		HTML:    html,
	}, nil
}

func (c *LiveClient) MoveMessage(ctx context.Context, uid string, toFolder string, fromFolder string) error {
	client, err := c.dialIMAP()
	if err != nil {
		return err
	}
	defer client.Logout()

	_, err = client.Select(fromFolder, &imap.SelectOptions{ReadOnly: false}).Wait()
	if err != nil {
		return fmt.Errorf("failed to select folder %s: %w", fromFolder, err)
	}

	parsedUID, err := strconv.ParseUint(uid, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid UID: %s", uid)
	}

	var uidSet imap.UIDSet
	uidSet.AddNum(imap.UID(parsedUID))

	_, err = client.Move(uidSet, toFolder).Wait()
	return err
}

func (c *LiveClient) SaveDraft(ctx context.Context, toEmail, subject, body string) error {
	client, err := c.dialIMAP()
	if err != nil {
		return err
	}
	defer client.Logout()

	rawMsg := BuildMIMEMessage(c.Username, toEmail, subject, body, "")

	// Append to 'Черновики' (Mail.ru default), fallback to 'Drafts'
	appendOpts := &imap.AppendOptions{
		Flags: []imap.Flag{imap.FlagDraft},
		Time:  time.Now(),
	}

	cmd := client.Append("Черновики", int64(len(rawMsg)), appendOpts)
	if _, err := cmd.Write(rawMsg); err == nil {
		if err := cmd.Close(); err == nil {
			return nil
		}
	}

	// Fallback to Drafts
	cmd2 := client.Append("Drafts", int64(len(rawMsg)), appendOpts)
	if _, err := cmd2.Write(rawMsg); err != nil {
		return err
	}
	return cmd2.Close()
}

// BuildMIMEMessage builds an RFC822 compliant message with optional attachment.
func BuildMIMEMessage(from, to, subject, body, attachmentPath string) []byte {
	var buf bytes.Buffer
	boundary := fmt.Sprintf("boundary_%d", time.Now().UnixNano())

	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: =?UTF-8?B?%s?=\r\n", base64.StdEncoding.EncodeToString([]byte(subject))))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString("MIME-Version: 1.0\r\n")

	if attachmentPath != "" && fileExists(attachmentPath) {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(body)
		buf.WriteString("\r\n\r\n")

		// Attachment
		attBytes, err := os.ReadFile(attachmentPath)
		if err == nil {
			filename := filepath.Base(attachmentPath)
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: application/octet-stream; name=\"%s\"\r\n", filename))
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", filename))
			
			encoded := base64.StdEncoding.EncodeToString(attBytes)
			for len(encoded) > 76 {
				buf.WriteString(encoded[:76] + "\r\n")
				encoded = encoded[76:]
			}
			buf.WriteString(encoded + "\r\n")
		}
		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(body)
		buf.WriteString("\r\n")
	}

	return buf.Bytes()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// SendEmail sends an email via SMTP over TLS on port 465 with 15s deadline.
func (c *LiveClient) SendEmail(ctx context.Context, toEmail, subject, body, attachmentPath string) error {
	port := "465"
	host := c.SMTPHost
	if strings.Contains(host, ":") {
		h, p, err := net.SplitHostPort(host)
		if err == nil {
			host = h
			port = p
		}
	}

	dialer := &net.Dialer{Timeout: c.Timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, port), &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("SMTP TLS dial failed: %w", err)
	}
	defer conn.Close()

	smtpClient, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client failed: %w", err)
	}
	defer smtpClient.Quit()

	auth := smtp.PlainAuth("", c.Username, c.Password, host)
	if err := smtpClient.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth failed: %w", err)
	}

	if err := smtpClient.Mail(c.Username); err != nil {
		return err
	}
	if err := smtpClient.Rcpt(toEmail); err != nil {
		return err
	}

	w, err := smtpClient.Data()
	if err != nil {
		return err
	}
	defer w.Close()

	msg := BuildMIMEMessage(c.Username, toEmail, subject, body, attachmentPath)
	_, err = w.Write(msg)
	return err
}
