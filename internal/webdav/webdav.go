package webdav

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// AllowedRoots returns the permissible roots for filesystem access containment.
func AllowedRoots() []string {
	if custom := os.Getenv("MAILRU_ALLOWED_ROOTS"); custom != "" {
		var roots []string
		for _, r := range strings.Split(custom, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				roots = append(roots, r)
			}
		}
		if len(roots) > 0 {
			return roots
		}
	}

	roots := []string{os.TempDir()}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		roots = append(roots, home)
	}

	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		roots = append(roots, cwd)
	}

	// Ecosystem backwards compatibility paths if present
	for _, p := range []string{"/tmp/workspace"} {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			roots = append(roots, p)
		}
	}

	// Deduplicate roots
	seen := make(map[string]bool)
	var deduped []string
	for _, r := range roots {
		clean := filepath.Clean(r)
		if !seen[clean] {
			seen[clean] = true
			deduped = append(deduped, clean)
		}
	}

	return deduped
}

// ValidateDownloadPath checks if a target local path falls within authorized roots.
func ValidateDownloadPath(localPath string, allowedRoots []string) (string, error) {
	absTarget, err := filepath.Abs(localPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	evalTarget, err := filepath.EvalSymlinks(absTarget)
	if err == nil {
		absTarget = evalTarget
	} else if os.IsNotExist(err) {
		// If the file or directory doesn't exist yet, recursively walk up
		// the directory tree until we find an existing path to evaluate.
		dir := filepath.Dir(absTarget)
		var evalDir string
		var dirErr error

		for {
			evalDir, dirErr = filepath.EvalSymlinks(dir)
			if dirErr == nil {
				break
			}
			if !os.IsNotExist(dirErr) || dir == "/" || dir == "." {
				break
			}
			dir = filepath.Dir(dir)
		}

		if dirErr == nil {
			// Compute the relative path from the found directory to the original target
			rel, _ := filepath.Rel(dir, absTarget)
			absTarget = filepath.Join(evalDir, rel)
		}
	}

	cleanTarget := filepath.Clean(absTarget)

	for _, root := range allowedRoots {
		evalRoot, err := filepath.EvalSymlinks(root)
		if err == nil {
			root = evalRoot
		}
		cleanRoot := filepath.Clean(root)
		rel, err := filepath.Rel(cleanRoot, cleanTarget)
		if err == nil && !strings.HasPrefix(rel, "..") && rel != "." && !strings.HasPrefix(rel, "/..") {
			return cleanTarget, nil
		}
		if cleanRoot == cleanTarget {
			return cleanTarget, nil
		}
	}

	return "", fmt.Errorf("Path traversal detected. Downloads are restricted to: %s", strings.Join(allowedRoots, ", "))
}

// Client provides WebDAV operations with strict timeouts and path validation.
type Client struct {
	BaseURL  string
	Username string
	Password string
	Client   *http.Client
}

// NewClient creates a new WebDAV client.
func NewClient(baseURL, username, password string, timeout time.Duration) (*Client, error) {
	if username == "" || password == "" {
		return nil, errors.New("WebDAV username and password must not be empty")
	}
	if baseURL == "" {
		baseURL = "https://webdav.cloud.mail.ru"
	}
	return &Client{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Username: username,
		Password: password,
		Client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) buildURL(remotePath string) string {
	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}
	return c.BaseURL + remotePath
}

func (c *Client) newRequest(ctx context.Context, method, remotePath string, body io.Reader) (*http.Request, error) {
	reqURL := c.buildURL(remotePath)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	return req, nil
}

// XML structures for PROPFIND
type multiStatus struct {
	XMLName   xml.Name       `xml:"multistatus"`
	Responses []propResponse `xml:"response"`
}

type propResponse struct {
	Href     string   `xml:"href"`
	PropStat propStat `xml:"propstat"`
}

type propStat struct {
	Prop prop `xml:"prop"`
}

type prop struct {
	DisplayName   string       `xml:"displayname"`
	ResourceType  resourceType `xml:"resourcetype"`
	ContentLength *int64       `xml:"getcontentlength"`
}

type resourceType struct {
	Collection *struct{} `xml:"collection"`
}

// ListDirectory lists contents of a WebDAV folder.
func (c *Client) ListDirectory(ctx context.Context, remotePath string) ([]string, error) {
	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}
	req, err := c.newRequest(ctx, "PROPFIND", remotePath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create PROPFIND request: %w", err)
	}
	req.Header.Set("Depth", "1")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PROPFIND failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("PROPFIND failed with status %d: %s", resp.StatusCode, string(body))
	}

	var ms multiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, fmt.Errorf("failed to decode WebDAV XML response: %w", err)
	}

	var items []string
	cleanRemote := strings.TrimRight(remotePath, "/") + "/"

	for _, r := range ms.Responses {
		decodedHref, err := url.PathUnescape(r.Href)
		if err != nil {
			decodedHref = r.Href
		}

		// Strip host/scheme if present
		if u, err := url.Parse(decodedHref); err == nil && u.Path != "" {
			decodedHref = u.Path
		}

		itemClean := strings.TrimRight(decodedHref, "/")
		if itemClean == strings.TrimRight(cleanRemote, "/") || decodedHref == cleanRemote || decodedHref == "/" {
			// Skip the requested directory itself
			continue
		}

		name := r.PropStat.Prop.DisplayName
		if name == "" {
			name = path.Base(decodedHref)
		}

		if r.PropStat.Prop.ResourceType.Collection != nil {
			items = append(items, name+"/")
		} else {
			items = append(items, name)
		}
	}

	return items, nil
}

// CreateDirectory creates a directory via MKCOL.
func (c *Client) CreateDirectory(ctx context.Context, remotePath string) error {
	req, err := c.newRequest(ctx, "MKCOL", remotePath, nil)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMethodNotAllowed {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create directory %s (status %d): %s", remotePath, resp.StatusCode, string(body))
	}
	return nil
}

// UploadFile uploads a local file to WebDAV via PUT.
func (c *Client) UploadFile(ctx context.Context, localPath, remotePath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer f.Close()

	req, err := c.newRequest(ctx, "PUT", remotePath, f)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload %s to %s (status %d): %s", localPath, remotePath, resp.StatusCode, string(body))
	}
	return nil
}

// DownloadFile downloads a file from WebDAV to a local path with path traversal defense.
func (c *Client) DownloadFile(ctx context.Context, remotePath, localPath string, allowedRoots []string) error {
	safePath, err := ValidateDownloadPath(localPath, allowedRoots)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(safePath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	req, err := c.newRequest(ctx, "GET", remotePath, nil)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to download %s (status %d): %s", remotePath, resp.StatusCode, string(body))
	}

	out, err := os.Create(safePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Delete removes a file or directory via DELETE.
func (c *Client) Delete(ctx context.Context, remotePath string) error {
	req, err := c.newRequest(ctx, "DELETE", remotePath, nil)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete %s (status %d): %s", remotePath, resp.StatusCode, string(body))
	}
	return nil
}
