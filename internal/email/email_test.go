package email_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rupivbluegreen/pactline/internal/email"
)

// isMailpitUnavailable detects when Mailpit's SMTP is not reachable, so
// the test can skip cleanly on machines without the dev compose stack.
func isMailpitUnavailable() bool {
	conn, err := net.DialTimeout("tcp", "localhost:1025", 500*time.Millisecond)
	if err != nil {
		return true
	}
	_ = conn.Close()
	return false
}

func TestSender_DeliversToMailpit(t *testing.T) {
	if isMailpitUnavailable() {
		t.Skip("mailpit unavailable on localhost:1025")
	}

	s := email.NewSender()
	to := "mailpit-test-" + t.Name() + "@example.com"
	subject := "pactline-test-" + t.Name()

	if err := s.Send(context.Background(), to, subject, "hello from test"); err != nil {
		t.Fatalf("send: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:8025/api/v1/search?query=" + subject)
		if err == nil {
			var body struct {
				Messages []struct {
					ID string `json:"ID"`
					To []struct {
						Address string `json:"Address"`
					} `json:"To"`
				} `json:"messages"`
			}
			err := json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
			if err == nil {
				for _, m := range body.Messages {
					for _, addr := range m.To {
						if strings.EqualFold(addr.Address, to) {
							return
						}
					}
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("message not found in mailpit")
}
