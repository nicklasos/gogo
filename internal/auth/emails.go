package auth

import (
	"bytes"
	"html/template"
	"net/url"

	"app/internal/db"
	"app/internal/mail"
)

// The emails are deliberately plain: one sentence, one link. Projects that need branding
// or translations replace these templates.
var emailHTML = template.Must(template.New("email").Parse(`<p>Hello {{.Name}},</p>
<p>{{.Intro}}</p>
<p><a href="{{.Link}}">{{.Action}}</a></p>
<p>{{.Outro}}</p>
<p>{{.App}}</p>
`))

type emailContent struct {
	App     string
	Name    string
	Subject string
	Intro   string
	Action  string
	Link    string
	Outro   string
}

func (c emailContent) message(to string) mail.Message {
	var html bytes.Buffer
	_ = emailHTML.Execute(&html, c)

	return mail.Message{
		To:      []string{to},
		Subject: c.Subject,
		Text:    "Hello " + c.Name + ",\n\n" + c.Intro + "\n\n" + c.Link + "\n\n" + c.Outro + "\n\n" + c.App + "\n",
		HTML:    html.String(),
	}
}

func (s *AuthService) link(path, token string) string {
	return s.opts.FrontendURL + path + "?token=" + url.QueryEscape(token)
}

func (s *AuthService) passwordResetEmail(user db.User, token string) mail.Message {
	return emailContent{
		App:     s.opts.AppName,
		Name:    user.Name,
		Subject: "Reset your " + s.opts.AppName + " password",
		Intro:   "We received a request to reset your password. Open the link below to choose a new one.",
		Action:  "Reset password",
		Link:    s.link("/reset-password", token),
		Outro:   "The link works once and expires in " + s.opts.PasswordResetTTL.String() + ". If you did not ask for it, ignore this email.",
	}.message(user.Email)
}

func (s *AuthService) emailVerificationEmail(user db.User, token string) mail.Message {
	return emailContent{
		App:     s.opts.AppName,
		Name:    user.Name,
		Subject: "Confirm your " + s.opts.AppName + " email address",
		Intro:   "Open the link below to confirm that this email address is yours.",
		Action:  "Confirm email",
		Link:    s.link("/verify-email", token),
		Outro:   "The link works once and expires in " + s.opts.EmailVerificationTTL.String() + ".",
	}.message(user.Email)
}
