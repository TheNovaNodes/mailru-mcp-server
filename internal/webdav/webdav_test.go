package webdav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWebDAV_Constructor(t *testing.T) {
	_, err := NewClient("https://webdav.cloud.mail.ru", "", "pass", 15*time.Second)
	if err == nil {
		t.Fatal("expected error on empty username")
	}

	cli, err := NewClient("", "user", "pass", 15*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli.BaseURL != "https://webdav.cloud.mail.ru" {
		t.Errorf("expected default base url, got %s", cli.BaseURL)
	}

	// buildURL without leading slash
	u := cli.buildURL("test/path")
	if u != "https://webdav.cloud.mail.ru/test/path" {
		t.Errorf("got %s", u)
	}
}

func TestWebDAV_PathTraversal(t *testing.T) {
	allowedRoots := []string{"/tmp/workspace"}

	// Valid path
	safe, err := ValidateDownloadPath("/tmp/workspace/file.pdf", allowedRoots)
	if err != nil {
		t.Fatalf("unexpected path validation error: %v", err)
	}
	if safe != "/tmp/workspace/file.pdf" {
		t.Errorf("got %s", safe)
	}

	// Exact root match
	safeRoot, err := ValidateDownloadPath("/tmp/workspace", allowedRoots)
	if err != nil || safeRoot != "/tmp/workspace" {
		t.Errorf("expected root match, got %s, err: %v", safeRoot, err)
	}

	// Traversal attempt to /etc/shadow
	_, err = ValidateDownloadPath("/tmp/workspace/../../etc/shadow", allowedRoots)
	if err == nil {
		t.Fatal("expected path traversal detection, but succeeded")
	}

	// Path outside allowed roots
	_, err = ValidateDownloadPath("/var/log/app.log", allowedRoots)
	if err == nil {
		t.Fatal("expected forbidden path outside roots, but succeeded")
	}
}

func TestWebDAV_AllowedRoots(t *testing.T) {
	roots := AllowedRoots()
	if len(roots) < 2 {
		t.Fatalf("expected at least 2 default roots, got %v", roots)
	}

	// Test custom MAILRU_ALLOWED_ROOTS
	t.Setenv("MAILRU_ALLOWED_ROOTS", "/custom/one, /custom/two")
	customRoots := AllowedRoots()
	if len(customRoots) != 2 || customRoots[0] != "/custom/one" || customRoots[1] != "/custom/two" {
		t.Fatalf("expected custom roots [/custom/one, /custom/two], got %v", customRoots)
	}
}

func TestWebDAV_ListDirectory(t *testing.T) {
	xmlPayload := `<?xml version="1.0" encoding="utf-8"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/docs/</href>
    <propstat>
      <prop>
        <displayname>docs</displayname>
        <resourcetype><collection/></resourcetype>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
  <response>
    <href>/docs/subfolder/</href>
    <propstat>
      <prop>
        <displayname></displayname>
        <resourcetype><collection/></resourcetype>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
  <response>
    <href>/docs/contract.pdf</href>
    <propstat>
      <prop>
        <displayname>contract.pdf</displayname>
        <resourcetype/>
        <getcontentlength>1024</getcontentlength>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			t.Errorf("expected PROPFIND, got %s", r.Method)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(xmlPayload))
	}))
	defer server.Close()

	cli, err := NewClient(server.URL, "testuser", "testpass", 5*time.Second)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	items, err := cli.ListDirectory(context.Background(), "docs")
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d: %v", len(items), items)
	}
	if items[0] != "subfolder/" {
		t.Errorf("expected subfolder/, got %s", items[0])
	}
	if items[1] != "contract.pdf" {
		t.Errorf("expected contract.pdf, got %s", items[1])
	}
}

func TestWebDAV_Operations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			w.WriteHeader(http.StatusCreated)
		case "PUT":
			w.WriteHeader(http.StatusCreated)
		case "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("mock-file-content"))
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	cli, _ := NewClient(server.URL, "testuser", "testpass", 5*time.Second)
	ctx := context.Background()

	// MKCOL without slash
	if err := cli.CreateDirectory(ctx, "new_folder"); err != nil {
		t.Fatalf("CreateDirectory failed: %v", err)
	}

	// PUT
	tmpFile := filepath.Join(t.TempDir(), "upload.txt")
	_ = os.WriteFile(tmpFile, []byte("hello"), 0644)
	if err := cli.UploadFile(ctx, tmpFile, "remote/upload.txt"); err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	// GET with path containment
	allowedRoot := t.TempDir()
	downloadTarget := filepath.Join(allowedRoot, "subdir", "downloaded.txt")
	if err := cli.DownloadFile(ctx, "remote/upload.txt", downloadTarget, []string{allowedRoot}); err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}

	content, err := os.ReadFile(downloadTarget)
	if err != nil || string(content) != "mock-file-content" {
		t.Fatalf("unexpected downloaded content: %s, err: %v", string(content), err)
	}

	// DELETE
	if err := cli.Delete(ctx, "remote/upload.txt"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestWebDAV_ErrorsAndEdgeCases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/badxml" {
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte("not xml"))
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}))
	serverURL := server.URL
	server.Close() // Closed server triggers network Do errors

	cli, _ := NewClient(serverURL, "user", "pass", 100*time.Millisecond)
	ctx := context.Background()

	// Network errors
	if _, err := cli.ListDirectory(ctx, "/"); err == nil {
		t.Error("expected error on closed server")
	}
	if err := cli.CreateDirectory(ctx, "/foo"); err == nil {
		t.Error("expected error on closed server")
	}
	tmpFile := filepath.Join(t.TempDir(), "up.txt")
	_ = os.WriteFile(tmpFile, []byte("x"), 0644)
	if err := cli.UploadFile(ctx, tmpFile, "/bar"); err == nil {
		t.Error("expected error on closed server")
	}
	if err := cli.DownloadFile(ctx, "/foo", tmpFile, []string{filepath.Dir(tmpFile)}); err == nil {
		t.Error("expected error on closed server")
	}
	if err := cli.Delete(ctx, "/foo"); err == nil {
		t.Error("expected error on closed server")
	}

	// Test upload missing local file
	if err := cli.UploadFile(ctx, "/nonexistent/file.bin", "/remote.bin"); err == nil {
		t.Error("expected error uploading missing file")
	}

	// Test download path traversal violation
	if err := cli.DownloadFile(ctx, "/remote.bin", "/etc/passwd", []string{"/tmp"}); err == nil {
		t.Error("expected path traversal violation")
	}

	// Test 500 status and XML decode error on running server
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/badxml" {
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte("invalid-xml"))
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
	}))
	defer srv2.Close()

	cli2, _ := NewClient(srv2.URL, "user", "pass", time.Second)
	if _, err := cli2.ListDirectory(ctx, "/badxml"); err == nil {
		t.Error("expected XML decode error")
	}
	if _, err := cli2.ListDirectory(ctx, "/500"); err == nil {
		t.Error("expected status 500 error")
	}
	if err := cli2.CreateDirectory(ctx, "/500"); err == nil {
		t.Error("expected status 500 error")
	}
	if err := cli2.UploadFile(ctx, tmpFile, "/500"); err == nil {
		t.Error("expected status 500 error")
	}
	if err := cli2.DownloadFile(ctx, "/500", tmpFile, []string{filepath.Dir(tmpFile)}); err == nil {
		t.Error("expected status 500 error")
	}
	if err := cli2.Delete(ctx, "/500"); err == nil {
		t.Error("expected status 500 error")
	}
}
