package main

import (
	"fmt"
	"log"
	"os"

	"github.com/TheNovaNodes/mailru-mcp-server/internal/config"
	"github.com/TheNovaNodes/mailru-mcp-server/internal/hitl"
	"github.com/TheNovaNodes/mailru-mcp-server/internal/mail"
	"github.com/TheNovaNodes/mailru-mcp-server/internal/server"
	"github.com/TheNovaNodes/mailru-mcp-server/internal/webdav"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}

	davCli, err := webdav.NewClient(cfg.WebDAVHost, cfg.Username, cfg.Password, cfg.Timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: WebDAV client init failed: %v\n", err)
		os.Exit(1)
	}

	mailCli, err := mail.NewLiveClient(cfg.Username, cfg.Password, cfg.IMAPHost, cfg.SMTPHost, cfg.Timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Mail client init failed: %v\n", err)
		os.Exit(1)
	}

	hitlMgr := hitl.NewManager(hitl.DefaultMaxPending, hitl.DefaultTTL)
	srv := server.NewServer(mailCli, davCli, hitlMgr, nil)

	if err := mcpserver.ServeStdio(srv.MCPServer()); err != nil {
		log.Fatalf("MCP Server terminated with error: %v", err)
	}
}
