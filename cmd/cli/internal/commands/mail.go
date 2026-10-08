package commands

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"app/cmd/cli/internal"
	"app/internal/mail"
)

// RunMail sends a live test email via the configured SMTP server.
func RunMail(app *internal.CLIApp, args []string) {
	fs := flag.NewFlagSet("mail", flag.ExitOnError)
	var (
		to      = fs.String("to", "", "Recipient email address (required)")
		subject = fs.String("subject", app.Config.AppName+" mail test", "Email subject")
		body    = fs.String("body", "Hello, this is a test email from "+app.Config.AppName+".", "Plain-text body")
	)
	fs.Usage = func() {
		fmt.Println("Usage: make cli -- mail [options]")
		fmt.Println()
		fmt.Println("Send a test email through the configured SMTP server.")
		fmt.Println("Requires MAIL_* env vars; see .env.example.")
		fmt.Println("The mail is sent even when APP_DEBUG=true.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println(`  make cli -- mail -to you@example.com`)
		fmt.Println(`  make cli -- mail -to you@example.com -subject "Hello" -body "It works"`)
	}
	fs.Parse(args)

	if *to == "" {
		fmt.Fprintln(os.Stderr, "error: -to is required")
		fs.Usage()
		os.Exit(1)
	}

	svc := mail.NewService(app.Config, app.Logger)
	if err := svc.SendLive(context.Background(), mail.Message{
		To:      []string{*to},
		Subject: *subject,
		Text:    *body,
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Mail sent to %s. Check the inbox.\n", *to)
}
