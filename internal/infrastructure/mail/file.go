package mail

import (
	"context"
	"fmt"
	"os"
)

// File is a development-only mail sink. Messages never enter the public web root.
type File struct{ Directory string }

func (f File) Send(ctx context.Context, to, subject, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(f.Directory, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(f.Directory, "message-*.txt")
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "To: %s\nSubject: %s\n\n%s\n", to, subject, body)
	return err
}

func (f File) SendMultipart(ctx context.Context, to, subject, text, html string) error {
	if err := f.Send(ctx, to, subject, text); err != nil {
		return err
	}
	file, err := os.CreateTemp(f.Directory, "message-*.html")
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(html)
	return err
}
