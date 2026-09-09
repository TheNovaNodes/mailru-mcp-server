package mail

import (
	"crypto/tls"
	"net"
	"testing"
	"bufio"
	"strings"
	"fmt"
)

// runMockIMAPServer starts a TLS listener and handles simple IMAP commands.
func runMockIMAPServer(t *testing.T, srvConfig *tls.Config, handler func(net.Conn, *bufio.Reader)) string {
	l, err := tls.Listen("tcp", "127.0.0.1:0", srvConfig)
	if err != nil {
		t.Fatalf("failed to start mock IMAP server: %v", err)
	}
	
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				c.Write([]byte("* OK [CAPABILITY IMAP4rev1] Mock IMAP Server Ready\r\n"))
				r := bufio.NewReader(c)
				if handler != nil {
					handler(c, r)
				}
			}(conn)
		}
	}()
	
	return l.Addr().String()
}

func readIMAPCommand(r *bufio.Reader) (string, string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	line = strings.TrimSpace(line)
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return parts[0], "", nil
	}
	return parts[0], parts[1], nil
}

// defaultIMAPAuthHandler handles CAPABILITY and LOGIN
func defaultIMAPAuthHandler(c net.Conn, r *bufio.Reader) error {
	for {
		tag, cmd, err := readIMAPCommand(r)
		if err != nil { return err }
		cmdUpper := strings.ToUpper(cmd)
		if strings.HasPrefix(cmdUpper, "CAPABILITY") {
			c.Write([]byte(fmt.Sprintf("* CAPABILITY IMAP4rev1\r\n%s OK CAPABILITY completed\r\n", tag)))
		} else if strings.HasPrefix(cmdUpper, "ENABLE") {
			c.Write([]byte(fmt.Sprintf("%s OK ENABLE completed\r\n", tag)))
		} else if strings.HasPrefix(cmdUpper, "LOGIN") {
			c.Write([]byte(fmt.Sprintf("%s OK User logged in\r\n", tag)))
			return nil
		} else {
			c.Write([]byte(fmt.Sprintf("%s OK Command ignored\r\n", tag)))
		}
	}
}
// runMockSMTPServer starts a TLS listener and handles simple SMTP commands.
func runMockSMTPServer(t *testing.T, srvConfig *tls.Config) string {
	l, err := tls.Listen("tcp", "127.0.0.1:0", srvConfig)
	if err != nil {
		t.Fatalf("failed to start mock SMTP server: %v", err)
	}
	
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				c.Write([]byte("220 mock-smtp\r\n"))
				r := bufio.NewReader(c)
				for {
					line, err := r.ReadString('\n')
					if err != nil { return }
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "EHLO") {
						c.Write([]byte("250-mock-smtp\r\n250 AUTH PLAIN\r\n"))
					} else if strings.HasPrefix(line, "AUTH PLAIN") {
						c.Write([]byte("235 Authentication successful\r\n"))
					} else if strings.HasPrefix(line, "MAIL FROM:") {
						c.Write([]byte("250 OK\r\n"))
					} else if strings.HasPrefix(line, "RCPT TO:") {
						if strings.Contains(line, "FAIL") {
							c.Write([]byte("550 No such user\r\n"))
						} else {
							c.Write([]byte("250 OK\r\n"))
						}
					} else if strings.HasPrefix(line, "DATA") {
						c.Write([]byte("354 Start mail input; end with <CRLF>.<CRLF>\r\n"))
						for {
							dataLine, err := r.ReadString('\n')
							if err != nil { return }
							if strings.TrimSpace(dataLine) == "." {
								c.Write([]byte("250 OK\r\n"))
								break
							}
						}
					} else if strings.HasPrefix(line, "QUIT") {
						c.Write([]byte("221 Bye\r\n"))
						return
					} else {
						c.Write([]byte("500 Error\r\n"))
					}
				}
			}(conn)
		}
	}()
	
	return l.Addr().String()
}
