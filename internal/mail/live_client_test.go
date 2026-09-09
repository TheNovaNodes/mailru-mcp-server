package mail

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// The helper functions runMockIMAPServer, runMockSMTPServer, readIMAPCommand are in mock_imap_smtp_test.go

func TestLiveClient_FetchRecentEmails(t *testing.T) {
	srvConfig, cleanup := setupMockTLS(t)
	defer cleanup()

	addr := runMockIMAPServer(t, srvConfig, func(c net.Conn, r *bufio.Reader) {
		for {
			tag, cmd, err := readIMAPCommand(r)
			if err != nil { return }
			upperCmd := strings.ToUpper(cmd)
			
			if strings.HasPrefix(upperCmd, "CAPABILITY") {
				c.Write([]byte(fmt.Sprintf("* CAPABILITY IMAP4rev1 ENABLE AUTH=PLAIN\r\n%s OK CAPABILITY completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "ENABLE") {
				c.Write([]byte(fmt.Sprintf("* ENABLE\r\n%s OK ENABLE completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "LOGIN") {
				c.Write([]byte(fmt.Sprintf("%s OK User logged in\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "EXAMINE") || strings.HasPrefix(upperCmd, "SELECT") {
				if strings.Contains(upperCmd, "INVALID_FOLDER") {
					c.Write([]byte(fmt.Sprintf("%s NO Folder not found\r\n", tag)))
				} else if strings.Contains(upperCmd, "EMPTY_FOLDER") {
					c.Write([]byte(fmt.Sprintf("* 0 EXISTS\r\n%s OK [READ-ONLY] SELECT completed\r\n", tag)))
				} else {
					c.Write([]byte(fmt.Sprintf("* 1 EXISTS\r\n%s OK [READ-ONLY] SELECT completed\r\n", tag)))
				}
			} else if strings.HasPrefix(upperCmd, "FETCH") {
				msg := "From: Sender <sender@mail.ru>\r\nTo: rcpt@mail.ru\r\nSubject: Test Subject\r\nDate: Mon, 2 Jan 2006 15:04:05 -0700\r\n\r\nBody text here"
				bodyLen := len(msg)
				c.Write([]byte(fmt.Sprintf("* 1 FETCH (UID 100 FLAGS (\\Seen) ENVELOPE (\"Mon, 2 Jan 2006 15:04:05 -0700\" \"Test Subject\" ((\"Sender\" NIL \"sender\" \"mail.ru\")) ((\"Sender\" NIL \"sender\" \"mail.ru\")) ((\"Sender\" NIL \"sender\" \"mail.ru\")) ((NIL NIL \"rcpt\" \"mail.ru\")) NIL NIL NIL \"<msg-id>\") BODY[] {%d}\r\n%s)\r\n", bodyLen, msg)))
				c.Write([]byte(fmt.Sprintf("%s OK FETCH completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "LOGOUT") {
				c.Write([]byte(fmt.Sprintf("* BYE Logging out\r\n%s OK Logout completed\r\n", tag)))
				return
			} else {
				c.Write([]byte(fmt.Sprintf("%s OK Defaulted\r\n", tag)))
			}
		}
	})

	client, err := NewLiveClient("test@mail.ru", "pass", addr, addr, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx := context.Background()

	tests := []struct {
		name        string
		folder      string
		expectedLen int
		wantErr     bool
	}{
		{"Invalid Folder", "INVALID_FOLDER", 0, false},
		{"Empty Folder", "EMPTY_FOLDER", 0, false},
		{"Valid Folder", "INBOX", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := client.FetchRecentEmails(ctx, 5, tt.folder, false)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchRecentEmails() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(res) != tt.expectedLen {
				t.Errorf("FetchRecentEmails() len = %d, expected %d", len(res), tt.expectedLen)
			}
			if len(res) > 0 && res[0].UID != "100" {
				t.Errorf("expected UID 100, got %s", res[0].UID)
			}
		})
	}
}

func TestLiveClient_SearchEmails(t *testing.T) {
	srvConfig, cleanup := setupMockTLS(t)
	defer cleanup()

	addr := runMockIMAPServer(t, srvConfig, func(c net.Conn, r *bufio.Reader) {
		for {
			tag, cmd, err := readIMAPCommand(r)
			if err != nil { return }
			upperCmd := strings.ToUpper(cmd)
			
			if strings.HasPrefix(upperCmd, "CAPABILITY") {
				c.Write([]byte(fmt.Sprintf("* CAPABILITY IMAP4rev1 ENABLE AUTH=PLAIN\r\n%s OK CAPABILITY completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "ENABLE") {
				c.Write([]byte(fmt.Sprintf("* ENABLE\r\n%s OK ENABLE completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "LOGIN") {
				c.Write([]byte(fmt.Sprintf("%s OK User logged in\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "EXAMINE") || strings.HasPrefix(upperCmd, "SELECT") {
				if strings.Contains(upperCmd, "INVALID") {
					c.Write([]byte(fmt.Sprintf("%s NO Folder not found\r\n", tag)))
				} else {
					c.Write([]byte(fmt.Sprintf("* 1 EXISTS\r\n%s OK [READ-ONLY] SELECT completed\r\n", tag)))
				}
			} else if strings.HasPrefix(upperCmd, "UID SEARCH") {
				if strings.Contains(upperCmd, "NO_RESULT") {
					c.Write([]byte(fmt.Sprintf("* SEARCH\r\n%s OK SEARCH completed\r\n", tag)))
				} else {
					c.Write([]byte(fmt.Sprintf("* SEARCH 100\r\n%s OK SEARCH completed\r\n", tag)))
				}
			} else if strings.HasPrefix(upperCmd, "UID FETCH") {
				msg := "From: Searcher <search@mail.ru>\r\nTo: rcpt@mail.ru\r\nSubject: Search Match\r\nDate: Mon, 2 Jan 2006 15:04:05 -0700\r\n\r\nMatch body text"
				bodyLen := len(msg)
				c.Write([]byte(fmt.Sprintf("* 1 FETCH (UID 100 ENVELOPE (\"Mon, 2 Jan 2006 15:04:05 -0700\" \"Search Match\" ((\"Searcher\" NIL \"search\" \"mail.ru\")) ((\"Searcher\" NIL \"search\" \"mail.ru\")) ((\"Searcher\" NIL \"search\" \"mail.ru\")) ((NIL NIL \"rcpt\" \"mail.ru\")) NIL NIL NIL \"<msg-id>\") BODY[] {%d}\r\n%s)\r\n", bodyLen, msg)))
				c.Write([]byte(fmt.Sprintf("%s OK FETCH completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "LOGOUT") {
				c.Write([]byte(fmt.Sprintf("* BYE Logging out\r\n%s OK Logout completed\r\n", tag)))
				return
			} else {
				c.Write([]byte(fmt.Sprintf("%s OK Defaulted\r\n", tag)))
			}
		}
	})

	client, err := NewLiveClient("test@mail.ru", "pass", addr, addr, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx := context.Background()

	tests := []struct {
		name        string
		query       string
		folder      string
		expectedLen int
		wantErr     bool
	}{
		{"Invalid Folder", "test", "INVALID", 0, true},
		{"No Result", "NO_RESULT", "INBOX", 0, false},
		{"Valid Search", "VALID", "INBOX", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := client.SearchEmails(ctx, tt.query, tt.folder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SearchEmails() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(res) != tt.expectedLen {
				t.Errorf("SearchEmails() len = %d, expected %d", len(res), tt.expectedLen)
			}
			if len(res) > 0 && res[0].UID != "100" {
				t.Errorf("expected UID 100, got %s", res[0].UID)
			}
		})
	}
}

func TestLiveClient_GetEmailBody(t *testing.T) {
	srvConfig, cleanup := setupMockTLS(t)
	defer cleanup()

	addr := runMockIMAPServer(t, srvConfig, func(c net.Conn, r *bufio.Reader) {
		for {
			tag, cmd, err := readIMAPCommand(r)
			if err != nil { return }
			upperCmd := strings.ToUpper(cmd)
			
			if strings.HasPrefix(upperCmd, "CAPABILITY") {
				c.Write([]byte(fmt.Sprintf("* CAPABILITY IMAP4rev1 ENABLE AUTH=PLAIN\r\n%s OK CAPABILITY completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "ENABLE") {
				c.Write([]byte(fmt.Sprintf("* ENABLE\r\n%s OK ENABLE completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "LOGIN") {
				c.Write([]byte(fmt.Sprintf("%s OK User logged in\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "EXAMINE") || strings.HasPrefix(upperCmd, "SELECT") {
				if strings.Contains(upperCmd, "INVALID") {
					c.Write([]byte(fmt.Sprintf("%s NO Folder not found\r\n", tag)))
				} else {
					c.Write([]byte(fmt.Sprintf("* 1 EXISTS\r\n%s OK [READ-ONLY] SELECT completed\r\n", tag)))
				}
			} else if strings.HasPrefix(upperCmd, "UID FETCH") {
				if strings.Contains(upperCmd, "999") {
					c.Write([]byte(fmt.Sprintf("%s OK FETCH completed\r\n", tag)))
				} else {
					msg := "From: Sender <sender@mail.ru>\r\nTo: rcpt@mail.ru\r\nSubject: Fetch Match\r\nDate: Mon, 2 Jan 2006 15:04:05 -0700\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nBody detailed text"
					bodyLen := len(msg)
					c.Write([]byte(fmt.Sprintf("* 1 FETCH (UID 100 ENVELOPE (\"Mon, 2 Jan 2006 15:04:05 -0700\" \"Fetch Match\" ((\"Sender\" NIL \"sender\" \"mail.ru\")) ((\"Sender\" NIL \"sender\" \"mail.ru\")) ((\"Sender\" NIL \"sender\" \"mail.ru\")) ((NIL NIL \"rcpt\" \"mail.ru\")) NIL NIL NIL \"<msg-id>\") BODY[] {%d}\r\n%s)\r\n", bodyLen, msg)))
					c.Write([]byte(fmt.Sprintf("%s OK FETCH completed\r\n", tag)))
				}
			} else if strings.HasPrefix(upperCmd, "LOGOUT") {
				c.Write([]byte(fmt.Sprintf("* BYE Logging out\r\n%s OK Logout completed\r\n", tag)))
				return
			} else {
				c.Write([]byte(fmt.Sprintf("%s OK Defaulted\r\n", tag)))
			}
		}
	})

	client, err := NewLiveClient("test@mail.ru", "pass", addr, addr, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx := context.Background()

	tests := []struct {
		name    string
		uid     string
		folder  string
		wantErr bool
	}{
		{"Invalid Folder", "100", "INVALID", true},
		{"Invalid UID Parse", "abc", "INBOX", true},
		{"UID Not Found", "999", "INBOX", true},
		{"Valid UID", "100", "INBOX", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := client.GetEmailBody(ctx, tt.uid, tt.folder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetEmailBody() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && res.UID != "100" {
				t.Errorf("expected UID 100, got %s", res.UID)
			}
		})
	}
}

func TestLiveClient_MoveMessage(t *testing.T) {
	srvConfig, cleanup := setupMockTLS(t)
	defer cleanup()

	addr := runMockIMAPServer(t, srvConfig, func(c net.Conn, r *bufio.Reader) {
		for {
			tag, cmd, err := readIMAPCommand(r)
			if err != nil { return }
			upperCmd := strings.ToUpper(cmd)
			
			if strings.HasPrefix(upperCmd, "CAPABILITY") {
				c.Write([]byte(fmt.Sprintf("* CAPABILITY IMAP4rev1 ENABLE AUTH=PLAIN\r\n%s OK CAPABILITY completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "ENABLE") {
				c.Write([]byte(fmt.Sprintf("* ENABLE\r\n%s OK ENABLE completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "LOGIN") {
				c.Write([]byte(fmt.Sprintf("%s OK User logged in\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "SELECT") {
				if strings.Contains(upperCmd, "INVALID") {
					c.Write([]byte(fmt.Sprintf("%s NO Folder not found\r\n", tag)))
				} else {
					c.Write([]byte(fmt.Sprintf("* 1 EXISTS\r\n%s OK [READ-WRITE] SELECT completed\r\n", tag)))
				}
			} else if strings.HasPrefix(upperCmd, "UID MOVE") {
				c.Write([]byte(fmt.Sprintf("%s OK MOVE completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "LOGOUT") {
				c.Write([]byte(fmt.Sprintf("* BYE Logging out\r\n%s OK Logout completed\r\n", tag)))
				return
			} else {
				c.Write([]byte(fmt.Sprintf("%s OK Defaulted\r\n", tag)))
			}
		}
	})

	client, err := NewLiveClient("test@mail.ru", "pass", addr, addr, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx := context.Background()

	tests := []struct {
		name       string
		uid        string
		toFolder   string
		fromFolder string
		wantErr    bool
	}{
		{"Invalid Source Folder", "100", "Trash", "INVALID", true},
		{"Invalid UID", "abc", "Trash", "INBOX", true},
		{"Successful Move", "100", "Trash", "INBOX", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.MoveMessage(ctx, tt.uid, tt.toFolder, tt.fromFolder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("MoveMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLiveClient_SaveDraft(t *testing.T) {
	srvConfig, cleanup := setupMockTLS(t)
	defer cleanup()

	addr := runMockIMAPServer(t, srvConfig, func(c net.Conn, r *bufio.Reader) {
		var lastAppendTag string
		for {
			line, err := r.ReadString('\n')
			if err != nil { return }
			line = strings.TrimSpace(line)
			parts := strings.SplitN(line, " ", 2)
			if len(parts) < 2 { continue }
			tag := parts[0]
			cmd := strings.ToUpper(parts[1])
			
			if strings.HasPrefix(cmd, "CAPABILITY") {
				c.Write([]byte(fmt.Sprintf("* CAPABILITY IMAP4rev1 ENABLE AUTH=PLAIN\r\n%s OK CAPABILITY completed\r\n", tag)))
			} else if strings.HasPrefix(cmd, "ENABLE") {
				c.Write([]byte(fmt.Sprintf("* ENABLE\r\n%s OK ENABLE completed\r\n", tag)))
			} else if strings.HasPrefix(cmd, "LOGIN") {
				c.Write([]byte(fmt.Sprintf("%s OK User logged in\r\n", tag)))
			} else if strings.HasPrefix(cmd, "APPEND") {
				if strings.Contains(cmd, "Черновики") {
					c.Write([]byte(fmt.Sprintf("%s NO Folder not found\r\n", tag)))
				} else {
					lastAppendTag = tag
					c.Write([]byte("+ Ready for literal\r\n"))
				}
			} else if strings.HasPrefix(cmd, "LOGOUT") {
				c.Write([]byte(fmt.Sprintf("* BYE Logging out\r\n%s OK Logout completed\r\n", tag)))
				return
			} else if lastAppendTag != "" {
				c.Write([]byte(fmt.Sprintf("%s OK APPEND completed\r\n", lastAppendTag)))
				lastAppendTag = ""
			} else {
				c.Write([]byte(fmt.Sprintf("%s OK Defaulted\r\n", tag)))
			}
		}
	})

	client, err := NewLiveClient("test@mail.ru", "pass", addr, addr, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx := context.Background()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{"Save with Fallback", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.SaveDraft(ctx, "a@mail.ru", "sub", "body")
			if (err != nil) != tt.wantErr {
				t.Fatalf("SaveDraft() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLiveClient_SendEmail(t *testing.T) {
	srvConfig, cleanup := setupMockTLS(t)
	defer cleanup()

	addr := runMockSMTPServer(t, srvConfig)

	client, err := NewLiveClient("test@mail.ru", "pass", "127.0.0.1:993", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx := context.Background()

	tests := []struct {
		name    string
		rcpt    string
		wantErr bool
	}{
		{"Successful Send", "rcpt@mail.ru", false},
		{"Failed Recipient", "FAIL@mail.ru", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.SendEmail(ctx, tt.rcpt, "Subj", "Body text", "")
			if (err != nil) != tt.wantErr {
				t.Fatalf("SendEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLiveClient_Failures(t *testing.T) {
	srvConfig, cleanup := setupMockTLS(t)
	defer cleanup()

	addr := runMockIMAPServer(t, srvConfig, func(c net.Conn, r *bufio.Reader) {
		for {
			tag, cmd, err := readIMAPCommand(r)
			if err != nil { return }
			upperCmd := strings.ToUpper(cmd)
			
			if strings.HasPrefix(upperCmd, "CAPABILITY") {
				c.Write([]byte(fmt.Sprintf("* CAPABILITY IMAP4rev1 ENABLE AUTH=PLAIN\r\n%s OK CAPABILITY completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "ENABLE") {
				c.Write([]byte(fmt.Sprintf("* ENABLE\r\n%s OK ENABLE completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "LOGIN") {
				if strings.Contains(upperCmd, "BADPASS") {
					c.Write([]byte(fmt.Sprintf("%s NO Login failed\r\n", tag)))
				} else if strings.Contains(upperCmd, "DISCONNECT") {
					c.Close()
					return
				} else if strings.Contains(upperCmd, "TIMEOUT") {
					time.Sleep(3 * time.Second) // Force a timeout
					c.Write([]byte(fmt.Sprintf("%s OK User logged in\r\n", tag)))
				} else {
					c.Write([]byte(fmt.Sprintf("%s OK User logged in\r\n", tag)))
				}
			} else if strings.HasPrefix(upperCmd, "LOGOUT") {
				c.Write([]byte(fmt.Sprintf("* BYE Logging out\r\n%s OK Logout completed\r\n", tag)))
				return
			} else {
				c.Write([]byte(fmt.Sprintf("%s OK Defaulted\r\n", tag)))
			}
		}
	})

	tests := []struct {
		name     string
		password string
		timeout  time.Duration
		wantErr  bool
	}{
		{"Auth Failure", "BADPASS", 2 * time.Second, true},
		{"Disconnect", "DISCONNECT", 2 * time.Second, true},
		{"Timeout", "TIMEOUT", 500 * time.Millisecond, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewLiveClient("test@mail.ru", tt.password, addr, addr, tt.timeout)
			if err != nil {
				t.Fatalf("unexpected NewLiveClient error: %v", err)
			}
			ctx := context.Background()
			
			// This will trigger dialing and thus the login failure / disconnect / timeout.
			ctx, cancel := context.WithTimeout(ctx, tt.timeout)
			defer cancel()
			_, err = client.FetchRecentEmails(ctx, 5, "INBOX", false)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchRecentEmails() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLiveClient_MIMEParsingErrors(t *testing.T) {
	srvConfig, cleanup := setupMockTLS(t)
	defer cleanup()

	addr := runMockIMAPServer(t, srvConfig, func(c net.Conn, r *bufio.Reader) {
		for {
			tag, cmd, err := readIMAPCommand(r)
			if err != nil { return }
			upperCmd := strings.ToUpper(cmd)
			
			if strings.HasPrefix(upperCmd, "CAPABILITY") {
				c.Write([]byte(fmt.Sprintf("* CAPABILITY IMAP4rev1 ENABLE AUTH=PLAIN\r\n%s OK CAPABILITY completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "ENABLE") {
				c.Write([]byte(fmt.Sprintf("* ENABLE\r\n%s OK ENABLE completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "LOGIN") {
				c.Write([]byte(fmt.Sprintf("%s OK User logged in\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "SELECT") {
				c.Write([]byte(fmt.Sprintf("* 1 EXISTS\r\n%s OK [READ-ONLY] SELECT completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "UID FETCH") {
				if strings.Contains(upperCmd, "101") {
					// Invalid MIME headers
					msg := "From: \r\nTo: \r\nInvalid-Header\r\n\r\n"
					bodyLen := len(msg)
					c.Write([]byte(fmt.Sprintf("* 1 FETCH (UID 101 ENVELOPE (\"\" \"\" NIL NIL NIL NIL NIL NIL NIL NIL) BODY[] {%d}\r\n%s)\r\n", bodyLen, msg)))
					c.Write([]byte(fmt.Sprintf("%s OK FETCH completed\r\n", tag)))
				} else if strings.Contains(upperCmd, "102") {
					// Malformed multipart
					msg := "Content-Type: multipart/mixed; boundary=\"bad-boundary\"\r\n\r\n--bad-boundary\r\nContent-Type: text/plain\r\n\r\nHello\r\n--bad-boundary\r\nContent-Type: image/jpeg\r\nContent-Transfer-Encoding: base64\r\n\r\n!!!not-base64!!!\r\n--bad-boundary--\r\n"
					bodyLen := len(msg)
					c.Write([]byte(fmt.Sprintf("* 1 FETCH (UID 102 ENVELOPE (\"\" \"\" NIL NIL NIL NIL NIL NIL NIL NIL) BODY[] {%d}\r\n%s)\r\n", bodyLen, msg)))
					c.Write([]byte(fmt.Sprintf("%s OK FETCH completed\r\n", tag)))
				}
			} else if strings.HasPrefix(upperCmd, "LOGOUT") {
				c.Write([]byte(fmt.Sprintf("* BYE Logging out\r\n%s OK Logout completed\r\n", tag)))
				return
			} else {
				c.Write([]byte(fmt.Sprintf("%s OK Defaulted\r\n", tag)))
			}
		}
	})

	client, err := NewLiveClient("test@mail.ru", "pass", addr, addr, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx := context.Background()

	tests := []struct {
		name string
		uid  string
	}{
		{"Invalid Headers", "101"},
		{"Malformed Multipart Attachment", "102"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := client.GetEmailBody(ctx, tt.uid, "INBOX")
			if err != nil {
				t.Fatalf("expected GetEmailBody to handle MIME errors gracefully but got error: %v", err)
			}
			if res.UID != tt.uid {
				t.Errorf("expected UID %s, got %s", tt.uid, res.UID)
			}
			// Should fallback gracefully without panic
		})
	}
}
