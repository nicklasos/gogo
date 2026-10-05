package mail

import (
	"context"
	"fmt"
	"log"
	"strings"

	"app/config"
	"app/internal/logger"

	gomail "github.com/wneessen/go-mail"
)

// Sender is what modules depend on, so tests can swap in a MemorySender.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

type Message struct {
	To      []string
	Subject string
	Text    string
	HTML    string
}

type Service struct {
	config *config.Config
	logger *logger.Logger
}

func NewService(cfg *config.Config, logg *logger.Logger) *Service {
	if cfg == nil {
		log.Fatal("config cannot be nil when creating mail Service")
	}
	if logg == nil {
		log.Fatal("logger cannot be nil when creating mail Service")
	}
	return &Service{config: cfg, logger: logg}
}

// Send skips the network when APP_DEBUG=true and logs the message instead, body included,
// so links in emails (password reset, verification) can be followed during development.
func (s *Service) Send(ctx context.Context, msg Message) error {
	if s.config.Debug {
		s.logger.InfoContext(ctx, "Debug mode: skipping mail send",
			"to", msg.To,
			"subject", msg.Subject,
			"text", msg.Text,
		)
		return nil
	}
	return s.SendLive(ctx, msg)
}

// SendLive always dials SMTP (e.g. CLI smoke test while APP_DEBUG=true).
func (s *Service) SendLive(ctx context.Context, msg Message) error {
	if err := s.validate(msg); err != nil {
		return err
	}

	m := gomail.NewMsg()
	fromAddr := strings.TrimSpace(s.config.MailFromAddress)
	fromName := strings.TrimSpace(s.config.MailFromName)
	if fromName != "" {
		if err := m.FromFormat(fromName, fromAddr); err != nil {
			return fmt.Errorf("mail: set from: %w", err)
		}
	} else {
		if err := m.From(fromAddr); err != nil {
			return fmt.Errorf("mail: set from: %w", err)
		}
	}

	for _, to := range msg.To {
		to = strings.TrimSpace(to)
		if to == "" {
			continue
		}
		if err := m.To(to); err != nil {
			return fmt.Errorf("mail: set to %q: %w", to, err)
		}
	}

	m.Subject(msg.Subject)
	if strings.TrimSpace(msg.Text) != "" {
		m.SetBodyString(gomail.TypeTextPlain, msg.Text)
	}
	if strings.TrimSpace(msg.HTML) != "" {
		if strings.TrimSpace(msg.Text) != "" {
			m.AddAlternativeString(gomail.TypeTextHTML, msg.HTML)
		} else {
			m.SetBodyString(gomail.TypeTextHTML, msg.HTML)
		}
	}

	client, err := s.newClient()
	if err != nil {
		return err
	}
	if err := client.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("mail: send failed: %w", err)
	}
	return nil
}

func (s *Service) validate(msg Message) error {
	if strings.TrimSpace(s.config.MailHost) == "" {
		return fmt.Errorf("mail: MAIL_HOST is not configured")
	}
	if strings.TrimSpace(s.config.MailFromAddress) == "" {
		return fmt.Errorf("mail: MAIL_FROM_ADDRESS is not configured")
	}
	if len(msg.To) == 0 {
		return fmt.Errorf("mail: at least one recipient is required")
	}
	hasRecipient := false
	for _, to := range msg.To {
		if strings.TrimSpace(to) != "" {
			hasRecipient = true
			break
		}
	}
	if !hasRecipient {
		return fmt.Errorf("mail: at least one recipient is required")
	}
	if strings.TrimSpace(msg.Subject) == "" {
		return fmt.Errorf("mail: subject is required")
	}
	if strings.TrimSpace(msg.Text) == "" && strings.TrimSpace(msg.HTML) == "" {
		return fmt.Errorf("mail: text or html body is required")
	}
	mailer := strings.ToLower(strings.TrimSpace(s.config.MailMailer))
	if mailer != "" && mailer != "smtp" {
		return fmt.Errorf("mail: unsupported MAIL_MAILER %q (only smtp is implemented)", s.config.MailMailer)
	}
	return nil
}

func (s *Service) newClient() (*gomail.Client, error) {
	host := strings.TrimSpace(s.config.MailHost)
	port := s.config.MailPort
	if port <= 0 {
		port = 587
	}

	opts := []gomail.Option{
		gomail.WithPort(port),
	}

	scheme := strings.ToLower(strings.TrimSpace(s.config.MailScheme))
	switch scheme {
	case "smtps":
		opts = append(opts, gomail.WithSSL())
	case "none":
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	default: // smtp
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	}

	user := strings.TrimSpace(s.config.MailUsername)
	pass := s.config.MailPassword
	if user != "" || pass != "" {
		auth := gomail.SMTPAuthPlain
		if scheme == "none" {
			auth = gomail.SMTPAuthPlainNoEnc
		}
		opts = append(opts,
			gomail.WithSMTPAuth(auth),
			gomail.WithUsername(user),
			gomail.WithPassword(pass),
		)
	}

	client, err := gomail.NewClient(host, opts...)
	if err != nil {
		return nil, fmt.Errorf("mail: create smtp client: %w", err)
	}
	return client, nil
}
