package mail

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	htmltemplate "html/template"
	texttemplate "text/template"
)

// Templates port the sibling Laravel Blade content/layout and fa.json strings.
//
//go:embed templates/*
var templates embed.FS

type Notifications struct{ Transport application.Mailer }

func (n Notifications) Send(ctx context.Context, to, subject, text string) error {
	return n.Transport.Send(ctx, to, subject, text)
}
func Render(notice application.Notification) (string, string, error) {
	if notice.Locale != "fa" {
		notice.Locale = "en"
	}
	if notice.Kind != "reset" && notice.Kind != "verify" {
		return "", "", fmt.Errorf("unknown notification kind")
	}
	if notice.AppName == "" {
		notice.AppName = "Laravel"
	}
	if notice.Name == "" {
		if notice.Locale == "fa" {
			notice.Name = "کاربر"
		} else {
			notice.Name = "User"
		}
	}
	base := "templates/" + notice.Kind + "." + notice.Locale
	h, err := htmltemplate.ParseFS(templates, base+".html")
	if err != nil {
		return "", "", err
	}
	t, err := texttemplate.ParseFS(templates, base+".txt")
	if err != nil {
		return "", "", err
	}
	var html, text bytes.Buffer
	if err = h.Execute(&html, notice); err != nil {
		return "", "", err
	}
	if err = t.Execute(&text, notice); err != nil {
		return "", "", err
	}
	return text.String(), html.String(), nil
}
func (n Notifications) SendNotification(ctx context.Context, to, subject string, notice application.Notification) error {
	text, html, err := Render(notice)
	if err != nil {
		return err
	}
	if rich, ok := n.Transport.(application.MultipartMailer); ok {
		return rich.SendMultipart(ctx, to, subject, text, html)
	}
	return n.Transport.Send(ctx, to, subject, text)
}
