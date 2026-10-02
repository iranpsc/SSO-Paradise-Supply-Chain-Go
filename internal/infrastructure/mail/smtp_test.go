package mail

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net"
	netmail "net/mail"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSMTPMultipartDeliveryToLocalFakeServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	received := make(chan []byte, 1)
	fail := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			fail <- err
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(5 * time.Second))
		writer := bufio.NewWriter(conn)
		reader := textproto.NewReader(bufio.NewReader(conn))
		reply := func(line string) { writer.WriteString(line + "\r\n"); writer.Flush() }
		reply("220 localhost test")
		for {
			line, err := reader.ReadLine()
			if err != nil {
				fail <- err
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				reply("250 localhost")
			case strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
				reply("250 OK")
			case line == "DATA":
				reply("354 send")
				data, err := reader.ReadDotBytes()
				if err != nil {
					fail <- err
					return
				}
				received <- data
				reply("250 accepted")
			case line == "QUIT":
				reply("221 bye")
				return
			default:
				reply("500 unsupported")
			}
		}
	}()
	host, rawPort, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(rawPort)
	smtp := SMTP{Host: host, Port: port, From: "accounts@example.com"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = smtp.SendMultipart(ctx, "recipient@example.com", "Verification", "Plain link", "<p>HTML link</p>"); err != nil {
		t.Fatal(err)
	}
	var raw []byte
	select {
	case raw = <-received:
	case err := <-fail:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	message, err := netmail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	kind, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil || kind != "multipart/alternative" {
		t.Fatal("missing alternative MIME")
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	for _, expected := range []string{"Plain link", "<p>HTML link</p>"} {
		part, err := reader.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, part))
		if err != nil || string(body) != expected {
			t.Fatal("MIME body damaged", err)
		}
	}
	if err = smtp.Send(ctx, "bad\r\nBcc: victim@example.com", "subject", "body"); err == nil {
		t.Fatal("header injection accepted")
	}
}
