package domain

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestDocumentLimitAndAvatarLimit(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	data := append(b.Bytes(), make([]byte, (1<<20)+100)...)
	if msg := ValidateDocument("card.png", data); msg != "" {
		t.Fatal(msg)
	}
	if msg := ValidateImage("avatar.png", data); msg == "" {
		t.Fatal("avatar limit raised unintentionally")
	}
	if msg := ValidateDocument("card.php.png", data); msg == "" {
		t.Fatal("unsafe original filename accepted")
	}
	if msg := ValidateDocument("card.png", append(data, make([]byte, 1<<20)...)); msg == "" {
		t.Fatal("oversized document accepted")
	}
	if msg := ValidateDocument("card.png", []byte("fake image")); msg == "" {
		t.Fatal("fake image accepted")
	}
}
