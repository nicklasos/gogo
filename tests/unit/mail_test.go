package unit

import (
	"bufio"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"app/config"
	"app/internal/logger"
	"app/internal/mail"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMailLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.New(logger.Config{Level: "error", Format: "text", Output: "stdout"})
	require.NoError(t, err)
	return log
}

func TestMailService_Send_skipsWhenDebug(t *testing.T) {
	t.Parallel()

	svc := mail.NewService(&config.Config{
		Debug:           true,
		MailHost:        "127.0.0.1",
		MailPort:        1, // would fail if dialed
		MailScheme:      "none",
		MailFromAddress: "noreply@example.com",
		MailFromName:    "MyApp",
	}, testMailLogger(t))

	err := svc.Send(context.Background(), mail.Message{
		To:      []string{"user@example.com"},
		Subject: "skip me",
		Text:    "body",
	})
	require.NoError(t, err)
}

func TestMailService_SendLive_missingHost(t *testing.T) {
	t.Parallel()

	svc := mail.NewService(&config.Config{
		MailFromAddress: "noreply@example.com",
	}, testMailLogger(t))

	err := svc.SendLive(context.Background(), mail.Message{
		To:      []string{"user@example.com"},
		Subject: "test",
		Text:    "body",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MAIL_HOST")
}

func TestMailService_SendLive_missingFrom(t *testing.T) {
	t.Parallel()

	svc := mail.NewService(&config.Config{
		MailHost: "smtp.example.com",
	}, testMailLogger(t))

	err := svc.SendLive(context.Background(), mail.Message{
		To:      []string{"user@example.com"},
		Subject: "test",
		Text:    "body",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MAIL_FROM_ADDRESS")
}

func TestMailService_SendLive_fakeSMTP(t *testing.T) {
	t.Parallel()

	captured, addr, cleanup := startFakeSMTP(t)
	defer cleanup()

	host, portStr, ok := strings.Cut(addr, ":")
	require.True(t, ok)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	svc := mail.NewService(&config.Config{
		MailHost:        host,
		MailPort:        port,
		MailScheme:      "none",
		MailFromAddress: "noreply@example.com",
		MailFromName:    "MyApp",
	}, testMailLogger(t))

	err = svc.SendLive(context.Background(), mail.Message{
		To:      []string{"user@example.com"},
		Subject: "Mail test",
		Text:    "Hello from the test",
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		captured.mu.Lock()
		defer captured.mu.Unlock()
		return captured.data != ""
	}, 2*time.Second, 20*time.Millisecond)

	captured.mu.Lock()
	defer captured.mu.Unlock()
	assert.Contains(t, captured.mailFrom, "noreply@example.com")
	assert.Contains(t, captured.rcptTo, "user@example.com")
	assert.Contains(t, captured.data, "Subject: Mail test")
	assert.Contains(t, captured.data, "Hello from the test")
	assert.Contains(t, captured.data, "From:")
}

type fakeSMTPCapture struct {
	mu       sync.Mutex
	mailFrom string
	rcptTo   string
	data     string
}

func startFakeSMTP(t *testing.T) (*fakeSMTPCapture, string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	cap := &fakeSMTPCapture{}
	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		handleFakeSMTP(conn, cap)
	}()

	cleanup := func() {
		_ = ln.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	return cap, ln.Addr().String(), cleanup
}

func handleFakeSMTP(conn net.Conn, cap *fakeSMTPCapture) {
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	write := func(line string) {
		_, _ = w.WriteString(line + "\r\n")
		_ = w.Flush()
	}

	write("220 localhost ESMTP test")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO"):
			write("250-localhost")
			write("250 OK")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			cap.mu.Lock()
			cap.mailFrom = line
			cap.mu.Unlock()
			write("250 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			cap.mu.Lock()
			cap.rcptTo = line
			cap.mu.Unlock()
			write("250 OK")
		case upper == "DATA":
			write("354 End data with <CR><LF>.<CR><LF>")
			var b strings.Builder
			for {
				dataLine, err := r.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						return
					}
					break
				}
				if dataLine == ".\r\n" || dataLine == ".\n" {
					break
				}
				b.WriteString(dataLine)
			}
			cap.mu.Lock()
			cap.data = b.String()
			cap.mu.Unlock()
			write("250 OK")
		case upper == "QUIT":
			write("221 Bye")
			return
		case upper == "RSET":
			write("250 OK")
		default:
			write("250 OK")
		}
	}
}

func TestMailService_SendLive_htmlWithTextAlternative(t *testing.T) {
	t.Parallel()

	captured, addr, cleanup := startFakeSMTP(t)
	defer cleanup()

	host, portStr, ok := strings.Cut(addr, ":")
	require.True(t, ok)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	svc := mail.NewService(&config.Config{
		MailHost:        host,
		MailPort:        port,
		MailScheme:      "none",
		MailFromAddress: "noreply@example.com",
	}, testMailLogger(t))

	err = svc.SendLive(context.Background(), mail.Message{
		To:      []string{"user@example.com"},
		Subject: "Both bodies",
		Text:    "plain body",
		HTML:    "<p>html body</p>",
	})
	require.NoError(t, err)

	captured.mu.Lock()
	defer captured.mu.Unlock()
	assert.Contains(t, captured.data, "multipart/alternative")
	assert.Contains(t, captured.data, "plain body")
	assert.Contains(t, captured.data, "<p>html body</p>")
}

func TestMailService_SendLive_validatesMessage(t *testing.T) {
	t.Parallel()

	svc := mail.NewService(&config.Config{
		MailHost:        "smtp.example.com",
		MailFromAddress: "noreply@example.com",
	}, testMailLogger(t))

	cases := map[string]mail.Message{
		"recipient": {Subject: "s", Text: "b"},
		"subject":   {To: []string{"user@example.com"}, Text: "b"},
		"body":      {To: []string{"user@example.com"}, Subject: "s"},
	}
	for want, msg := range cases {
		err := svc.SendLive(context.Background(), msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), want)
	}
}

func TestMemorySender_collectsMessages(t *testing.T) {
	t.Parallel()

	sender := mail.NewMemorySender()
	var _ mail.Sender = sender

	require.NoError(t, sender.Send(context.Background(), mail.Message{To: []string{"a@example.com"}, Subject: "first"}))
	require.NoError(t, sender.Send(context.Background(), mail.Message{To: []string{"b@example.com"}, Subject: "second"}))

	sent := sender.Sent()
	require.Len(t, sent, 2)
	assert.Equal(t, "first", sent[0].Subject)
	assert.Equal(t, []string{"b@example.com"}, sent[1].To)
}
