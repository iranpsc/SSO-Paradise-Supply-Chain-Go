package mail

import (
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/application"
	"strings"
	"testing"
)

func TestPortedTemplatesRenderSafely(t *testing.T) {
	for _, locale := range []string{"fa", "en"} {
		for _, kind := range []string{"verify", "reset"} {
			text, html, err := Render(application.Notification{Kind: kind, Locale: locale, Name: `<script>alert(1)</script>`, AppName: "SSO", Link: "https://sso.example/email?token=abc&expires=123", Minutes: 60, Year: 2026})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(text, "https://sso.example/") || !strings.Contains(html, "https://sso.example/") {
				t.Fatal("missing action URL")
			}
			if strings.Contains(html, "<script>alert") || strings.Contains(html, "ZgotmplZ") || strings.Contains(html, "{!!") || strings.Contains(text, "@php") {
				t.Fatal("unsafe or incomplete template")
			}
			if locale == "fa" && !strings.Contains(html, `dir="rtl"`) {
				t.Fatal("Persian template must be RTL")
			}
		}
	}
}
