package mail

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
)

// TestMock_IMAP_StoreSeenFlag verifies the IMAP STORE command for \Seen flag in mock environment
func TestMock_IMAP_StoreSeenFlag(t *testing.T) {
	srvConfig, cleanup := setupMockTLS(t)
	defer cleanup()

	var storeCommandReceived string

	addr := runMockIMAPServer(t, srvConfig, func(c net.Conn, r *bufio.Reader) {
		for {
			tag, cmd, err := readIMAPCommand(r)
			if err != nil {
				return
			}
			upperCmd := strings.ToUpper(cmd)

			if strings.HasPrefix(upperCmd, "CAPABILITY") {
				c.Write([]byte(fmt.Sprintf("* CAPABILITY IMAP4rev1 ENABLE AUTH=PLAIN\r\n%s OK CAPABILITY completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "ENABLE") {
				c.Write([]byte(fmt.Sprintf("* ENABLE\r\n%s OK ENABLE completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "LOGIN") {
				c.Write([]byte(fmt.Sprintf("%s OK User logged in\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "SELECT") {
				c.Write([]byte(fmt.Sprintf("* 1 EXISTS\r\n* FLAGS (\\Seen \\Answered \\Flagged \\Deleted \\Draft)\r\n* OK [PERMANENTFLAGS (\\Seen \\*)] Permanent flags\r\n* OK [UIDVALIDITY 12345] UIDs valid\r\n%s OK [READ-WRITE] SELECT completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "UID STORE") || strings.HasPrefix(upperCmd, "STORE") {
				storeCommandReceived = cmd
				c.Write([]byte(fmt.Sprintf("* 1 FETCH (UID 172579 FLAGS (\\Seen))\r\n%s OK STORE completed\r\n", tag)))
			} else if strings.HasPrefix(upperCmd, "UID FETCH") || strings.HasPrefix(upperCmd, "FETCH") {
				c.Write([]byte(fmt.Sprintf("* 1 FETCH (UID 172579 FLAGS (\\Seen))\r\n%s OK FETCH completed\r\n", tag)))
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
		t.Fatalf("unexpected error creating LiveClient: %v", err)
	}

	imapCli, err := client.dialIMAP()
	if err != nil {
		t.Fatalf("dialIMAP failed: %v", err)
	}
	defer func() {
		_ = imapCli.Logout().Wait()
		_ = imapCli.Close()
	}()

	// Select folder in Read-Write mode
	_, err = imapCli.Select("INBOX/News", &imap.SelectOptions{ReadOnly: false}).Wait()
	if err != nil {
		t.Fatalf("Select INBOX/News failed: %v", err)
	}

	var uidSet imap.UIDSet
	uidSet.AddNum(imap.UID(172579))

	// Execute STORE command to add \Seen
	storeFlags := &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagSeen},
	}
	fetchCmd := imapCli.Store(uidSet, storeFlags, nil)
	msgs, err := fetchCmd.Collect()
	if err != nil {
		t.Fatalf("Store command failed: %v", err)
	}

	if len(msgs) == 0 {
		t.Fatalf("expected at least 1 updated message response, got 0")
	}

	t.Logf("STORE command successfully executed. Server received: %s", storeCommandReceived)
	t.Logf("Updated flags for UID 172579: %v", msgs[0].Flags)

	hasSeen := false
	for _, fl := range msgs[0].Flags {
		if fl == imap.FlagSeen {
			hasSeen = true
			break
		}
	}
	if !hasSeen {
		t.Errorf("expected \\Seen flag in message response, got %v", msgs[0].Flags)
	}
}

// TestLive_IMAP_StoreSeenFlag executes a live smoke test against imap.mail.ru if credentials exist
func TestLive_IMAP_StoreSeenFlag(t *testing.T) {
	username := os.Getenv("MAILRU_USERNAME")
	password := os.Getenv("MAILRU_APP_PASS")
	if username == "" || password == "" {
		t.Skip("skipping live test: MAILRU_USERNAME and MAILRU_APP_PASS not set in environment")
	}

	client, err := NewLiveClient(username, password, "imap.mail.ru", "smtp.mail.ru", 10*time.Second)
	if err != nil {
		t.Fatalf("failed to create LiveClient: %v", err)
	}

	imapCli, err := client.dialIMAP()
	if err != nil {
		t.Fatalf("live dialIMAP failed: %v", err)
	}
	defer func() {
		_ = imapCli.Logout().Wait()
		_ = imapCli.Close()
	}()

	// 1. Select INBOX/News in Read-Write mode
	sel, err := imapCli.Select("INBOX/News", &imap.SelectOptions{ReadOnly: false}).Wait()
	if err != nil {
		t.Fatalf("failed to select INBOX/News: %v", err)
	}
	t.Logf("Selected INBOX/News: %d messages exist", sel.NumMessages)

	if sel.NumMessages == 0 {
		t.Skip("INBOX/News is empty, skipping live store test")
	}

	// 2. Fetch the latest message UID and current flags
	var seqSet imap.SeqSet
	seqSet.AddNum(sel.NumMessages) // latest sequence number

	fetchOpts := &imap.FetchOptions{
		UID:   true,
		Flags: true,
	}
	fetchMsgs, err := imapCli.Fetch(seqSet, fetchOpts).Collect()
	if err != nil || len(fetchMsgs) == 0 {
		t.Fatalf("failed to fetch latest message: %v", err)
	}

	targetUID := fetchMsgs[0].UID
	initialFlags := fetchMsgs[0].Flags
	t.Logf("Target message UID: %d, Initial Flags: %v", targetUID, initialFlags)

	// 3. Mark as \Seen via Store
	var uidSet imap.UIDSet
	uidSet.AddNum(targetUID)

	storeCmd := imapCli.Store(uidSet, &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagSeen},
	}, nil)

	storeRes, err := storeCmd.Collect()
	if err != nil {
		t.Fatalf("live Store (+Seen) failed: %v", err)
	}
	t.Logf("Store (+Seen) completed. Server response message count: %d", len(storeRes))

	// 4. Verify by re-fetching flags
	verifyMsgs, err := imapCli.Fetch(uidSet, &imap.FetchOptions{UID: true, Flags: true}).Collect()
	if err != nil || len(verifyMsgs) == 0 {
		t.Fatalf("failed to verify flags: %v", err)
	}

	updatedFlags := verifyMsgs[0].Flags
	t.Logf("Verified Updated Flags for UID %d: %v", targetUID, updatedFlags)

	hasSeen := false
	for _, fl := range updatedFlags {
		if fl == imap.FlagSeen {
			hasSeen = true
			break
		}
	}

	if !hasSeen {
		t.Fatalf("Verification failed: \\Seen flag was not set on server! Flags: %v", updatedFlags)
	}

	t.Logf("SMOKE TEST SUCCESS: \\Seen flag was successfully stored and verified on imap.mail.ru for UID %d", targetUID)
}
