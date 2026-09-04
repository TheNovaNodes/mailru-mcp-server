package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/TheNovaNodes/mailru-mcp-server/internal/hitl"
	"github.com/TheNovaNodes/mailru-mcp-server/internal/mail"
	"github.com/TheNovaNodes/mailru-mcp-server/internal/webdav"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Server coordinates MCP tool dispatching, HITL policies, and protocol clients.
type Server struct {
	mcpServer    *mcpserver.MCPServer
	mailCli      mail.Client
	davCli       *webdav.Client
	hitlMgr      *hitl.Manager
	allowedRoots []string
}

// NewServer registers all 13 tools and returns a ready-to-serve Server instance.
func NewServer(mailCli mail.Client, davCli *webdav.Client, hitlMgr *hitl.Manager, allowedRoots []string) *Server {
	if allowedRoots == nil {
		allowedRoots = webdav.AllowedRoots()
	}

	mcpSrv := mcpserver.NewMCPServer("mailru-mcp-server", "2.0.0")

	s := &Server{
		mcpServer:    mcpSrv,
		mailCli:      mailCli,
		davCli:       davCli,
		hitlMgr:      hitlMgr,
		allowedRoots: allowedRoots,
	}

	s.registerTools()
	return s
}

// MCPServer returns the underlying MCP server.
func (s *Server) MCPServer() *mcpserver.MCPServer {
	return s.mcpServer
}

func (s *Server) registerTools() {
	// 1. execute_pending_action
	s.mcpServer.AddTool(
		mcp.NewTool("execute_pending_action",
			mcp.WithDescription("Execute a destructive action that was previously blocked by HITL. The agent must retrieve the token from the blocked action response."),
			mcp.WithString("token", mcp.Required(), mcp.Description("One-time security token from the blocked action response")),
		),
		s.handleExecutePendingAction,
	)

	// 2. mail_read_inbox
	s.mcpServer.AddTool(
		mcp.NewTool("mail_read_inbox",
			mcp.WithDescription("Fetch latest emails and threads. If folder is INBOX and include_smart_folders is True, aggregates across INBOX and Mail.ru smart subfolders (Newsletters, Social, News, Receipts) sorted newest first."),
			mcp.WithNumber("limit", mcp.Description("Number of emails to fetch (default: 10, max: 50)")),
			mcp.WithString("folder", mcp.Description("IMAP folder name (default: 'INBOX')")),
			mcp.WithBoolean("include_smart_folders", mcp.Description("Whether to aggregate smart subfolders (default: true)")),
		),
		s.handleMailReadInbox,
	)

	// 3. mail_get_body
	s.mcpServer.AddTool(
		mcp.NewTool("mail_get_body",
			mcp.WithDescription("Extract full email text and html content by UID for AI summarization and analysis."),
			mcp.WithString("uid", mcp.Required(), mcp.Description("IMAP Message UID")),
			mcp.WithString("folder", mcp.Description("IMAP folder name (default: 'INBOX')")),
		),
		s.handleMailGetBody,
	)

	// 4. mail_send_draft
	s.mcpServer.AddTool(
		mcp.NewTool("mail_send_draft",
			mcp.WithDescription("Save an email draft for ZavLab to review (No HITL required)."),
			mcp.WithString("to_email", mcp.Required(), mcp.Description("Recipient email address")),
			mcp.WithString("subject", mcp.Required(), mcp.Description("Draft subject line")),
			mcp.WithString("body", mcp.Required(), mcp.Description("Draft message body")),
		),
		s.handleMailSendDraft,
	)

	// 5. mail_move_message
	s.mcpServer.AddTool(
		mcp.NewTool("mail_move_message",
			mcp.WithDescription("Move an email to another folder like 'Archive', 'SPAM', or 'REPLY_REQUIRED' (HITL protected)."),
			mcp.WithString("uid", mcp.Required(), mcp.Description("IMAP Message UID")),
			mcp.WithString("to_folder", mcp.Required(), mcp.Description("Destination folder name")),
			mcp.WithString("from_folder", mcp.Description("Source folder name (default: 'INBOX')")),
		),
		s.handleMailMoveMessage,
	)

	// 6. mail_search_thread
	s.mcpServer.AddTool(
		mcp.NewTool("mail_search_thread",
			mcp.WithDescription("Semantic and text search across mailboxes."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search keyword or expression")),
			mcp.WithString("folder", mcp.Description("IMAP folder name (default: 'INBOX')")),
			mcp.WithNumber("limit", mcp.Description("Max search results (default: 20, max: 50)")),
		),
		s.handleMailSearchThread,
	)

	// 7. mail_send_reply
	s.mcpServer.AddTool(
		mcp.NewTool("mail_send_reply",
			mcp.WithDescription("Send a direct reply to a client (HITL protected)."),
			mcp.WithString("to_email", mcp.Required(), mcp.Description("Recipient email address")),
			mcp.WithString("subject", mcp.Required(), mcp.Description("Email subject line")),
			mcp.WithString("body", mcp.Required(), mcp.Description("Email message body")),
		),
		s.handleMailSendReply,
	)

	// 8. mail_send_with_attachment
	s.mcpServer.AddTool(
		mcp.NewTool("mail_send_with_attachment",
			mcp.WithDescription("Send email with files retrieved from WebDAV/local storage (HITL protected)."),
			mcp.WithString("to_email", mcp.Required(), mcp.Description("Recipient email address")),
			mcp.WithString("subject", mcp.Required(), mcp.Description("Email subject line")),
			mcp.WithString("body", mcp.Required(), mcp.Description("Email message body")),
			mcp.WithString("attachment_path", mcp.Required(), mcp.Description("Absolute local filesystem path to attachment")),
		),
		s.handleMailSendWithAttachment,
	)

	// 9. dav_list_dir
	s.mcpServer.AddTool(
		mcp.NewTool("dav_list_dir",
			mcp.WithDescription("Read CRM folder structures and file metadata. Uses pagination."),
			mcp.WithString("path", mcp.Description("WebDAV remote directory path (default: '/')")),
			mcp.WithNumber("offset", mcp.Description("Pagination offset index (default: 0)")),
			mcp.WithNumber("limit", mcp.Description("Number of items per page (default: 50, max: 100)")),
		),
		s.handleDavListDir,
	)

	// 10. dav_create_folder
	s.mcpServer.AddTool(
		mcp.NewTool("dav_create_folder",
			mcp.WithDescription("Scaffold a new client directory (HITL protected)."),
			mcp.WithString("path", mcp.Required(), mcp.Description("Remote WebDAV directory path to create")),
		),
		s.handleDavCreateFolder,
	)

	// 11. dav_upload_file
	s.mcpServer.AddTool(
		mcp.NewTool("dav_upload_file",
			mcp.WithDescription("Upload documents to WebDAV (HITL protected)."),
			mcp.WithString("local_path", mcp.Required(), mcp.Description("Local source file path")),
			mcp.WithString("remote_path", mcp.Required(), mcp.Description("Remote WebDAV destination path")),
		),
		s.handleDavUploadFile,
	)

	// 12. dav_download_file
	s.mcpServer.AddTool(
		mcp.NewTool("dav_download_file",
			mcp.WithDescription("Download documents to agent workspace or local storage. Safe read operation."),
			mcp.WithString("remote_path", mcp.Required(), mcp.Description("Remote WebDAV file path")),
			mcp.WithString("local_path", mcp.Required(), mcp.Description("Local destination file path")),
		),
		s.handleDavDownloadFile,
	)

	// 13. dav_delete_file
	s.mcpServer.AddTool(
		mcp.NewTool("dav_delete_file",
			mcp.WithDescription("Delete documents or folders. (HITL protected)."),
			mcp.WithString("path", mcp.Required(), mcp.Description("Remote WebDAV path to delete")),
		),
		s.handleDavDeleteFile,
	)
}

func (s *Server) handleExecutePendingAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	token := req.GetString("token", "")
	if token == "" {
		return mcp.NewToolResultError("❌ Error: Invalid, expired, or already executed HITL token."), nil
	}

	act, err := s.hitlMgr.Pop(token)
	if err != nil {
		return mcp.NewToolResultError("❌ Error: " + err.Error()), nil
	}

	switch act.Type {
	case "mail_send":
		to, _ := act.Details["to"].(string)
		subj, _ := act.Details["subject"].(string)
		body, _ := act.Details["body"].(string)
		att, _ := act.Details["attachment"].(string)
		if err := s.mailCli.SendEmail(ctx, to, subj, body, att); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Execution failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("✅ Action Executed: Email sent to %s.", to)), nil

	case "mail_move_message":
		uid, _ := act.Details["uid"].(string)
		toFolder, _ := act.Details["to_folder"].(string)
		fromFolder, _ := act.Details["from_folder"].(string)
		if fromFolder == "" {
			fromFolder = "INBOX"
		}
		if err := s.mailCli.MoveMessage(ctx, uid, toFolder, fromFolder); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Execution failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("✅ Action Executed: Message %s moved to %s.", uid, toFolder)), nil

	case "dav_delete":
		remotePath, _ := act.Details["path"].(string)
		if err := s.davCli.Delete(ctx, remotePath); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Execution failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("✅ Action Executed: %s deleted.", remotePath)), nil

	case "dav_upload":
		localPath, _ := act.Details["local_path"].(string)
		remotePath, _ := act.Details["remote_path"].(string)
		if err := s.davCli.UploadFile(ctx, localPath, remotePath); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Execution failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("✅ Action Executed: File uploaded to %s.", remotePath)), nil

	case "dav_create_folder":
		remotePath, _ := act.Details["path"].(string)
		if err := s.davCli.CreateDirectory(ctx, remotePath); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("❌ Execution failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("✅ Action Executed: Folder %s created.", remotePath)), nil

	default:
		return mcp.NewToolResultError(fmt.Sprintf("❌ Unknown action type: %s", act.Type)), nil
	}
}

func (s *Server) handleMailReadInbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := req.GetInt("limit", 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	folder := req.GetString("folder", "INBOX")
	includeSmart := req.GetBool("include_smart_folders", true)

	emails, err := s.mailCli.FetchRecentEmails(ctx, limit, folder, includeSmart)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error reading inbox: %v", err)), nil
	}
	if len(emails) == 0 {
		return mcp.NewToolResultText("No emails found."), nil
	}

	var parts []string
	for _, e := range emails {
		fldStr := ""
		if e.Folder != "" {
			fldStr = fmt.Sprintf(" | Folder: %s", e.Folder)
		}
		flags := strings.Join(e.Flags, ", ")
		snip := e.Text
		if len(snip) > 200 {
			snip = snip[:200]
		}
		parts = append(parts, fmt.Sprintf("UID: %s%s | From: %s | Subject: %s\nDate: %s | Flags: %s\nSnippet: %s...",
			e.UID, fldStr, e.From, e.Subject, e.Date, flags, snip))
	}

	return mcp.NewToolResultText(strings.Join(parts, "\n\n")), nil
}

func (s *Server) handleMailGetBody(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uid := req.GetString("uid", "")
	folder := req.GetString("folder", "INBOX")

	details, err := s.mailCli.GetEmailBody(ctx, uid, folder)
	if err != nil || details == nil {
		return mcp.NewToolResultText(fmt.Sprintf("Email with UID %s not found in folder %s.", uid, folder)), nil
	}

	body := details.Text
	if body == "" {
		body = details.HTML
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("UID: %s | Folder: %s\n", details.UID, details.Folder))
	sb.WriteString(fmt.Sprintf("From: %s | To: %s\n", details.From, details.To))
	sb.WriteString(fmt.Sprintf("Subject: %s\nDate: %s\n", details.Subject, details.Date))
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(body)

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleMailSendDraft(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	to := req.GetString("to_email", "")
	subj := req.GetString("subject", "")
	body := req.GetString("body", "")

	if err := s.mailCli.SaveDraft(ctx, to, subj, body); err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error saving draft: %v", err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("✅ Draft to '%s' saved successfully in Drafts/Черновики.", to)), nil
}

func (s *Server) handleMailMoveMessage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uid := req.GetString("uid", "")
	toFolder := req.GetString("to_folder", "")
	fromFolder := req.GetString("from_folder", "INBOX")

	notice := s.hitlMgr.Request("mail_move_message", map[string]any{
		"uid":         uid,
		"to_folder":   toFolder,
		"from_folder": fromFolder,
	})
	return mcp.NewToolResultText(notice), nil
}

func (s *Server) handleMailSearchThread(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	folder := req.GetString("folder", "INBOX")
	limit := req.GetInt("limit", 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	items, err := s.mailCli.SearchEmails(ctx, query, folder)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error searching emails: %v", err)), nil
	}
	if len(items) == 0 {
		return mcp.NewToolResultText("No emails found matching query."), nil
	}

	var parts []string
	count := len(items)
	if count > limit {
		count = limit
	}

	for _, e := range items[:count] {
		parts = append(parts, fmt.Sprintf("UID: %s | From: %s | Subject: %s\nDate: %s\nSnippet: %s",
			e.UID, e.From, e.Subject, e.Date, e.TextSnippet))
	}

	out := strings.Join(parts, "\n\n")
	if len(items) > limit {
		out += fmt.Sprintf("\n\n... (Truncated. Found %d emails, showing first %d. Be more specific!)", len(items), limit)
	}

	return mcp.NewToolResultText(out), nil
}

func (s *Server) handleMailSendReply(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	to := req.GetString("to_email", "")
	subj := req.GetString("subject", "")
	body := req.GetString("body", "")

	notice := s.hitlMgr.Request("mail_send", map[string]any{
		"to":      to,
		"subject": subj,
		"body":    body,
	})
	return mcp.NewToolResultText(notice), nil
}

func (s *Server) handleMailSendWithAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	to := req.GetString("to_email", "")
	subj := req.GetString("subject", "")
	body := req.GetString("body", "")
	att := req.GetString("attachment_path", "")

	notice := s.hitlMgr.Request("mail_send", map[string]any{
		"to":         to,
		"subject":    subj,
		"body":       body,
		"attachment": att,
	})
	return mcp.NewToolResultText(notice), nil
}

func (s *Server) handleDavListDir(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := req.GetString("path", "/")
	offset := req.GetInt("offset", 0)
	limit := req.GetInt("limit", 50)
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	contents, err := s.davCli.ListDirectory(ctx, p)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("WebDAV list failed: %v", err)), nil
	}

	total := len(contents)
	paginated := contents
	if offset > total {
		paginated = nil
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		paginated = contents[offset:end]
	}

	header := fmt.Sprintf("📁 Directory: %s (Showing %d to %d of %d total items)\n----------------------------------------\n",
		p, offset, offset+len(paginated), total)

	if total == 0 {
		return mcp.NewToolResultText(header + "Directory is empty."), nil
	}

	return mcp.NewToolResultText(header + strings.Join(paginated, "\n")), nil
}

func (s *Server) handleDavCreateFolder(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := req.GetString("path", "")
	notice := s.hitlMgr.Request("dav_create_folder", map[string]any{"path": p})
	return mcp.NewToolResultText(notice), nil
}

func (s *Server) handleDavUploadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	localPath := req.GetString("local_path", "")
	remotePath := req.GetString("remote_path", "")
	notice := s.hitlMgr.Request("dav_upload", map[string]any{
		"local_path":  localPath,
		"remote_path": remotePath,
	})
	return mcp.NewToolResultText(notice), nil
}

func (s *Server) handleDavDownloadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	remotePath := req.GetString("remote_path", "")
	localPath := req.GetString("local_path", "")

	if _, err := webdav.ValidateDownloadPath(localPath, s.allowedRoots); err != nil {
		return mcp.NewToolResultText("❌ Security Error: " + err.Error()), nil
	}

	if err := s.davCli.DownloadFile(ctx, remotePath, localPath, s.allowedRoots); err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("❌ Download failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("✅ File downloaded to %s successfully.", localPath)), nil
}

func (s *Server) handleDavDeleteFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := req.GetString("path", "")
	notice := s.hitlMgr.Request("dav_delete", map[string]any{"path": p})
	return mcp.NewToolResultText(notice), nil
}
