package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TheNovaNodes/mailru-mcp-server/internal/hitl"
	"github.com/TheNovaNodes/mailru-mcp-server/internal/mail"
	"github.com/TheNovaNodes/mailru-mcp-server/internal/webdav"
	"github.com/mark3labs/mcp-go/mcp"
)

func setupTestServer(t *testing.T) (*Server, *mail.MockClient, *httptest.Server, string) {
	mockMail := mail.NewMockClient()

	mockDavServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/empty" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><multistatus xmlns="DAV:"><response><href>/empty/</href><propstat><prop><displayname>empty</displayname><resourcetype><collection/></resourcetype></prop></propstat></response></multistatus>`))
			return
		}
		if r.URL.Path == "/faileverything" {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		switch r.Method {
		case "PROPFIND":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<multistatus xmlns="DAV:">
  <response><href>/item1.txt</href><propstat><prop><displayname>item1.txt</displayname><resourcetype/></prop></propstat></response>
</multistatus>`))
		case "MKCOL":
			w.WriteHeader(http.StatusCreated)
		case "PUT":
			w.WriteHeader(http.StatusCreated)
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("file-content"))
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "error", http.StatusInternalServerError)
		}
	}))

	davCli, err := webdav.NewClient(mockDavServer.URL, "user", "pass", 5*time.Second)
	if err != nil {
		t.Fatalf("failed to create dav client: %v", err)
	}

	hitlMgr := hitl.NewManager(10, time.Hour)
	tmpDir := t.TempDir()

	srv := NewServer(mockMail, davCli, hitlMgr, []string{tmpDir})
	return srv, mockMail, mockDavServer, tmpDir
}

func callTool(s *Server, name string, args map[string]any) (string, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	ctx := context.Background()
	var res *mcp.CallToolResult
	var err error

	switch name {
	case "execute_pending_action":
		res, err = s.handleExecutePendingAction(ctx, req)
	case "mail_read_inbox":
		res, err = s.handleMailReadInbox(ctx, req)
	case "mail_get_body":
		res, err = s.handleMailGetBody(ctx, req)
	case "mail_send_draft":
		res, err = s.handleMailSendDraft(ctx, req)
	case "mail_move_message":
		res, err = s.handleMailMoveMessage(ctx, req)
	case "mail_search_thread":
		res, err = s.handleMailSearchThread(ctx, req)
	case "mail_send_reply":
		res, err = s.handleMailSendReply(ctx, req)
	case "mail_send_with_attachment":
		res, err = s.handleMailSendWithAttachment(ctx, req)
	case "dav_list_dir":
		res, err = s.handleDavListDir(ctx, req)
	case "dav_create_folder":
		res, err = s.handleDavCreateFolder(ctx, req)
	case "dav_upload_file":
		res, err = s.handleDavUploadFile(ctx, req)
	case "dav_download_file":
		res, err = s.handleDavDownloadFile(ctx, req)
	case "dav_delete_file":
		res, err = s.handleDavDeleteFile(ctx, req)
	}

	if err != nil {
		return "", err
	}
	if len(res.Content) > 0 {
		if textContent, ok := res.Content[0].(mcp.TextContent); ok {
			return textContent.Text, nil
		}
	}
	return "", nil
}

func extractToken(resp string) string {
	idx := strings.Index(resp, "token='")
	if idx == -1 {
		return ""
	}
	return resp[idx+7 : idx+7+32]
}

func TestServer_AllTools(t *testing.T) {
	srv, mockMail, davMock, tmpDir := setupTestServer(t)
	defer davMock.Close()

	if srv.MCPServer() == nil {
		t.Fatal("expected non-nil MCP server")
	}

	// 1. mail_read_inbox
	longText := strings.Repeat("A", 300)
	mockMail.Emails = []mail.EmailSummary{
		{UID: "1", Folder: "INBOX", Subject: "Hello", From: "a@b.com", Flags: []string{"\\Seen"}, Text: longText},
	}
	out, err := callTool(srv, "mail_read_inbox", map[string]any{"limit": float64(100)})
	if err != nil || !strings.Contains(out, "UID: 1") {
		t.Fatalf("mail_read_inbox failed: %s, err: %v", out, err)
	}

	// mail_read_inbox empty
	mockMail.Emails = nil
	out, _ = callTool(srv, "mail_read_inbox", map[string]any{})
	if out != "No emails found." {
		t.Errorf("expected 'No emails found.', got %s", out)
	}

	// 2. mail_get_body
	mockMail.Details["1"] = &mail.EmailDetails{
		UID:     "1",
		Folder:  "INBOX",
		From:    "sender@mail.ru",
		To:      "recv@mail.ru",
		Subject: "Subj",
		Date:    "2026-09-04",
		Text:    "Content line",
	}
	out, err = callTool(srv, "mail_get_body", map[string]any{"uid": "1"})
	if err != nil || !strings.Contains(out, "Content line") {
		t.Fatalf("mail_get_body failed: %s", out)
	}

	// mail_get_body with HTML fallback
	mockMail.Details["2"] = &mail.EmailDetails{
		UID:  "2",
		HTML: "<b>HTML body</b>",
	}
	out, _ = callTool(srv, "mail_get_body", map[string]any{"uid": "2"})
	if !strings.Contains(out, "HTML body") {
		t.Errorf("expected HTML fallback, got %s", out)
	}

	// mail_get_body not found
	out, _ = callTool(srv, "mail_get_body", map[string]any{"uid": "999"})
	if !strings.Contains(out, "not found") {
		t.Errorf("expected not found, got %s", out)
	}

	// 3. mail_send_draft
	out, err = callTool(srv, "mail_send_draft", map[string]any{
		"to_email": "x@mail.ru", "subject": "S", "body": "B",
	})
	if err != nil || !strings.Contains(out, "saved successfully") {
		t.Fatalf("mail_send_draft failed: %s", out)
	}

	// 4. mail_search_thread with truncation
	var searchItems []mail.EmailSearchItem
	for i := 1; i <= 25; i++ {
		searchItems = append(searchItems, mail.EmailSearchItem{
			UID: "1", Folder: "INBOX", Subject: "Invoice", From: "b@mail.ru", TextSnippet: "snip",
		})
	}
	mockMail.SearchItems = searchItems
	out, err = callTool(srv, "mail_search_thread", map[string]any{"query": "Invoice", "limit": float64(10)})
	if err != nil || !strings.Contains(out, "Truncated") {
		t.Fatalf("mail_search_thread truncation failed: %s", out)
	}

	// mail_search_thread empty
	out, _ = callTool(srv, "mail_search_thread", map[string]any{"query": "Nonexistent"})
	if !strings.Contains(out, "No emails found") {
		t.Errorf("expected no emails, got %s", out)
	}

	// 5. mail_send_reply (HITL)
	out, _ = callTool(srv, "mail_send_reply", map[string]any{"to_email": "c@mail.ru", "subject": "S", "body": "B"})
	tok := extractToken(out)
	if tok == "" {
		t.Fatalf("expected HITL token from mail_send_reply: %s", out)
	}
	execOut, _ := callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(execOut, "Action Executed: Email sent to c@mail.ru.") {
		t.Fatalf("execute_pending_action failed: %s", execOut)
	}

	// 6. mail_send_with_attachment (HITL)
	out, _ = callTool(srv, "mail_send_with_attachment", map[string]any{
		"to_email": "d@mail.ru", "subject": "S", "body": "B", "attachment_path": "/tmp/a.pdf",
	})
	tok = extractToken(out)
	execOut, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(execOut, "Action Executed: Email sent to d@mail.ru.") {
		t.Fatalf("execute mail_send_with_attachment failed: %s", execOut)
	}

	// 7. mail_move_message (HITL)
	out, _ = callTool(srv, "mail_move_message", map[string]any{"uid": "1", "to_folder": "Trash"})
	tok = extractToken(out)
	execOut, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(execOut, "Action Executed: Message 1 moved to Trash.") {
		t.Fatalf("execute mail_move_message failed: %s", execOut)
	}

	// 8. dav_list_dir with pagination and empty dir
	out, err = callTool(srv, "dav_list_dir", map[string]any{"path": "/", "offset": float64(0), "limit": float64(10)})
	if err != nil || !strings.Contains(out, "item1.txt") {
		t.Fatalf("dav_list_dir failed: %s", out)
	}
	out, err = callTool(srv, "dav_list_dir", map[string]any{"path": "/empty"})
	if err != nil || !strings.Contains(out, "Directory is empty.") {
		t.Fatalf("dav_list_dir empty failed: %s", out)
	}
	out, _ = callTool(srv, "dav_list_dir", map[string]any{"path": "/faileverything"})
	if !strings.Contains(out, "WebDAV list failed") {
		t.Fatalf("dav_list_dir error check failed: %s", out)
	}

	// 9. dav_create_folder (HITL)
	out, _ = callTool(srv, "dav_create_folder", map[string]any{"path": "/newdir"})
	tok = extractToken(out)
	execOut, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(execOut, "Action Executed: Folder /newdir created.") {
		t.Fatalf("execute dav_create_folder failed: %s", execOut)
	}

	// 10. dav_upload_file (HITL)
	locFile := filepath.Join(tmpDir, "up.txt")
	_ = os.WriteFile(locFile, []byte("xyz"), 0644)
	out, _ = callTool(srv, "dav_upload_file", map[string]any{"local_path": locFile, "remote_path": "/remote.txt"})
	tok = extractToken(out)
	execOut, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(execOut, "Action Executed: File uploaded to /remote.txt.") {
		t.Fatalf("execute dav_upload_file failed: %s", execOut)
	}

	// 11. dav_download_file
	downFile := filepath.Join(tmpDir, "down.txt")
	out, err = callTool(srv, "dav_download_file", map[string]any{"remote_path": "/remote.txt", "local_path": downFile})
	if err != nil || !strings.Contains(out, "File downloaded to") {
		t.Fatalf("dav_download_file failed: %s", out)
	}

	// dav_download_file failure from WebDAV
	out, _ = callTool(srv, "dav_download_file", map[string]any{"remote_path": "/faileverything", "local_path": filepath.Join(tmpDir, "fail.txt")})
	if !strings.Contains(out, "Download failed") {
		t.Fatalf("expected download failure, got: %s", out)
	}

	// dav_download_file path traversal attempt
	out, _ = callTool(srv, "dav_download_file", map[string]any{"remote_path": "/remote.txt", "local_path": "/etc/passwd"})
	if !strings.Contains(out, "Security Error") {
		t.Fatalf("expected security error on path traversal, got: %s", out)
	}

	// 12. dav_delete_file (HITL)
	out, _ = callTool(srv, "dav_delete_file", map[string]any{"path": "/del.txt"})
	tok = extractToken(out)
	execOut, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(execOut, "Action Executed: /del.txt deleted.") {
		t.Fatalf("execute dav_delete_file failed: %s", execOut)
	}

	// 13. execute_pending_action invalid token
	out, _ = callTool(srv, "execute_pending_action", map[string]any{"token": "invalid"})
	if !strings.Contains(out, "Error: Invalid, expired, or already executed") {
		t.Errorf("expected invalid token error, got %s", out)
	}
	out, _ = callTool(srv, "execute_pending_action", map[string]any{"token": ""})
	if !strings.Contains(out, "Error: Invalid, expired, or already executed") {
		t.Errorf("expected invalid token error, got %s", out)
	}
}

func TestServer_ErrorHandlingAndExecutionFailures(t *testing.T) {
	srv, mockMail, davMock, _ := setupTestServer(t)
	defer davMock.Close()

	// Default constructor with nil allowedRoots
	defaultSrv := NewServer(mockMail, srv.davCli, srv.hitlMgr, nil)
	if defaultSrv == nil || len(defaultSrv.allowedRoots) == 0 {
		t.Error("expected default allowed roots populated")
	}

	mockMail.ShouldFail = true
	out, _ := callTool(srv, "mail_read_inbox", map[string]any{})
	if !strings.Contains(out, "Error reading inbox") {
		t.Errorf("expected error, got %s", out)
	}

	out, _ = callTool(srv, "mail_search_thread", map[string]any{"query": "q"})
	if !strings.Contains(out, "Error searching emails") {
		t.Errorf("expected error, got %s", out)
	}

	out, _ = callTool(srv, "mail_send_draft", map[string]any{"to_email": "a", "subject": "b", "body": "c"})
	if !strings.Contains(out, "Error saving draft") {
		t.Errorf("expected error, got %s", out)
	}

	// Test execute_pending_action when mail action fails
	resp := srv.hitlMgr.Request("mail_send", map[string]any{"to": "err@mail.ru", "subject": "s", "body": "b"})
	tok := extractToken(resp)
	out, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(out, "Execution failed") {
		t.Errorf("expected execution failed, got: %s", out)
	}

	// Test execute_pending_action when move message fails
	resp = srv.hitlMgr.Request("mail_move_message", map[string]any{"uid": "1", "to_folder": "Archive"})
	tok = extractToken(resp)
	out, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(out, "Execution failed") {
		t.Errorf("expected execution failed, got: %s", out)
	}

	// Test execute_pending_action when dav actions fail
	resp = srv.hitlMgr.Request("dav_delete", map[string]any{"path": "/faileverything"})
	tok = extractToken(resp)
	out, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(out, "Execution failed") {
		t.Errorf("expected dav_delete failure, got: %s", out)
	}

	resp = srv.hitlMgr.Request("dav_create_folder", map[string]any{"path": "/faileverything"})
	tok = extractToken(resp)
	out, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(out, "Execution failed") {
		t.Errorf("expected dav_create_folder failure, got: %s", out)
	}

	resp = srv.hitlMgr.Request("dav_upload", map[string]any{"local_path": "/nonexistent", "remote_path": "/faileverything"})
	tok = extractToken(resp)
	out, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(out, "Execution failed") {
		t.Errorf("expected dav_upload failure, got: %s", out)
	}

	// Test execute_pending_action unknown type
	resp = srv.hitlMgr.Request("unknown_type", map[string]any{})
	tok = extractToken(resp)
	out, _ = callTool(srv, "execute_pending_action", map[string]any{"token": tok})
	if !strings.Contains(out, "Unknown action type") {
		t.Errorf("expected unknown action type, got: %s", out)
	}
}
