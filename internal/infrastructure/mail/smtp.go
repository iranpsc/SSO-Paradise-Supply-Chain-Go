package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

type SMTP struct {
	Host                                 string
	Port                                 int
	Username, Password, From, Encryption string
}

func (s SMTP) Send(ctx context.Context, to, subject, body string) error {
	return s.SendMultipart(ctx, to, subject, body, "")
}
func (s SMTP) SendMultipart(ctx context.Context, to, subject, body, html string) error {
	if strings.ContainsAny(to+subject+s.From, "\r\n") {
		return errors.New("mail headers contain line breaks")
	}
	recipient, err := mail.ParseAddress(to)
	if err != nil {
		return err
	}
	sender, err := mail.ParseAddress(s.From)
	if err != nil {
		return err
	}
	port := s.Port
	if port == 0 {
		port = 587
	}
	address := net.JoinHostPort(s.Host, strconv.Itoa(port))
	tlsConfig := &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	if s.Encryption == "ssl" {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", address)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(20 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err = conn.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	switch s.Encryption {
	case "tls":
		if err = client.StartTLS(tlsConfig); err != nil {
			return err
		}
	case "ssl", "":
	default:
		return errors.New("MAIL_ENCRYPTION must be tls, ssl or empty")
	}
	if s.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return err
		}
	}
	if err = client.Mail(sender.Address); err != nil {
		return err
	}
	if err = client.Rcpt(recipient.Address); err != nil {
		return err
	}
	data, err := client.Data()
	if err != nil {
		return err
	}
	// Base64 content avoids SMTP line-length and UTF-8 transport differences.
	payload := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s\r\n", sender.String(), recipient.String(), mime.QEncoding.Encode("utf-8", subject), encodedBody(body))
	if html != "" {
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		for _, part := range []struct{ kind, body string }{{"text/plain", body}, {"text/html", html}} {
			header := textproto.MIMEHeader{}
			header.Set("Content-Type", part.kind+"; charset=UTF-8")
			header.Set("Content-Transfer-Encoding", "base64")
			stream, e := writer.CreatePart(header)
			if e != nil {
				return e
			}
			if _, e = stream.Write([]byte(encodedBody(part.body))); e != nil {
				return e
			}
		}
		if err = writer.Close(); err != nil {
			return err
		}
		payload = fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n%s", sender.String(), recipient.String(), mime.QEncoding.Encode("utf-8", subject), writer.Boundary(), buffer.String())
	}
	if _, err = data.Write([]byte(payload)); err != nil {
		data.Close()
		return err
	}
	if err = data.Close(); err != nil {
		return err
	}
	return client.Quit()
}
func encodedBody(body string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	var lines []string
	for len(encoded) > 76 {
		lines = append(lines, encoded[:76])
		encoded = encoded[76:]
	}
	lines = append(lines, encoded)
	return strings.Join(lines, "\r\n")
}
