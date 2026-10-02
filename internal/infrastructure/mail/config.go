package mail

import (
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
)

func SMTPFromEnv() (SMTP, error) {
	port := 587
	if raw := os.Getenv("MAIL_PORT"); raw != "" {
		var err error
		port, err = strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return SMTP{}, fmt.Errorf("invalid MAIL_PORT")
		}
	}
	encryption := os.Getenv("MAIL_ENCRYPTION")
	if encryption == "" {
		encryption = "tls"
	}
	s := SMTP{Host: os.Getenv("MAIL_HOST"), Port: port, Username: os.Getenv("MAIL_USERNAME"), Password: os.Getenv("MAIL_PASSWORD"), From: os.Getenv("MAIL_FROM_ADDRESS"), Encryption: encryption}
	if s.Host == "" || strings.ContainsAny(s.Host+s.From, "\r\n") {
		return s, fmt.Errorf("MAIL_HOST and a valid MAIL_FROM_ADDRESS are required")
	}
	if _, err := mail.ParseAddress(s.From); err != nil {
		return s, fmt.Errorf("invalid MAIL_FROM_ADDRESS")
	}
	if name := os.Getenv("MAIL_FROM_NAME"); name != "" {
		if strings.ContainsAny(name, "\r\n") {
			return s, fmt.Errorf("invalid MAIL_FROM_NAME")
		}
		address, _ := mail.ParseAddress(s.From)
		s.From = (&mail.Address{Name: name, Address: address.Address}).String()
	}
	if s.Encryption != "tls" && s.Encryption != "ssl" && s.Encryption != "none" {
		return s, fmt.Errorf("MAIL_ENCRYPTION must be tls, ssl or none")
	}
	if s.Encryption == "none" {
		s.Encryption = ""
	}
	return s, nil
}
