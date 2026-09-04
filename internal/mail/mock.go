package mail

import (
	"context"
	"errors"
	"strings"
)

// MockClient implements Client for tests.
type MockClient struct {
	Emails          []EmailSummary
	SearchItems     []EmailSearchItem
	Details         map[string]*EmailDetails
	MovedMessages   map[string]string
	SavedDrafts     []map[string]string
	SentEmails      []map[string]string
	ShouldFail      bool
	FailError       error
}

func NewMockClient() *MockClient {
	return &MockClient{
		Details:       make(map[string]*EmailDetails),
		MovedMessages: make(map[string]string),
	}
}

func (m *MockClient) FetchRecentEmails(ctx context.Context, limit int, folder string, includeSmartFolders bool) ([]EmailSummary, error) {
	if m.ShouldFail {
		if m.FailError != nil {
			return nil, m.FailError
		}
		return nil, errors.New("mock fetch error")
	}
	res := m.Emails
	if len(res) > limit {
		res = res[:limit]
	}
	return res, nil
}

func (m *MockClient) SearchEmails(ctx context.Context, query string, folder string) ([]EmailSearchItem, error) {
	if m.ShouldFail {
		return nil, errors.New("mock search error")
	}
	var res []EmailSearchItem
	for _, item := range m.SearchItems {
		if strings.Contains(strings.ToLower(item.Subject), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(item.TextSnippet), strings.ToLower(query)) {
			res = append(res, item)
		}
	}
	return res, nil
}

func (m *MockClient) GetEmailBody(ctx context.Context, uid string, folder string) (*EmailDetails, error) {
	if m.ShouldFail {
		return nil, errors.New("mock get body error")
	}
	d, ok := m.Details[uid]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

func (m *MockClient) MoveMessage(ctx context.Context, uid string, toFolder string, fromFolder string) error {
	if m.ShouldFail {
		return errors.New("mock move error")
	}
	m.MovedMessages[uid] = toFolder
	return nil
}

func (m *MockClient) SaveDraft(ctx context.Context, toEmail, subject, body string) error {
	if m.ShouldFail {
		return errors.New("mock save draft error")
	}
	m.SavedDrafts = append(m.SavedDrafts, map[string]string{
		"to":      toEmail,
		"subject": subject,
		"body":    body,
	})
	return nil
}

func (m *MockClient) SendEmail(ctx context.Context, toEmail, subject, body, attachmentPath string) error {
	if m.ShouldFail {
		return errors.New("mock send error")
	}
	m.SentEmails = append(m.SentEmails, map[string]string{
		"to":         toEmail,
		"subject":    subject,
		"body":       body,
		"attachment": attachmentPath,
	})
	return nil
}
