package domain

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"path/filepath"
	"strings"
)

const maxImageBytes = 1 << 20 // 1024KB, matches Laravel max:1024

var allowedImageMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

var allowedImageExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

var dangerousExt = map[string]bool{
	".php": true, ".php3": true, ".php4": true, ".php5": true, ".phtml": true,
	".pht": true, ".phar": true, ".shtml": true, ".htaccess": true, ".exe": true,
	".sh": true, ".bat": true, ".cmd": true, ".js": true, ".html": true,
	".htm": true, ".svg": true, ".swf": true, ".xml": true,
}

// ValidateImage mirrors Laravel SecureImage (simplified): extension,
// magic bytes via DetectContentType, decodability for jpeg/png, 1MB cap.
func ValidateImage(filename string, data []byte) string {
	if len(data) == 0 {
		return "فایل معتبر نیست."
	}
	if len(data) > maxImageBytes {
		return "حجم فایل نباید بیشتر از ۱ مگابایت باشد."
	}
	lower := strings.ToLower(strings.TrimSpace(filename))
	if strings.Contains(lower, "\x00") || strings.Contains(lower, "%00") {
		return "نام فایل معتبر نیست."
	}
	ext := strings.ToLower(filepath.Ext(lower))
	if dangerousExt[ext] {
		return "پسوند فایل مجاز نیست."
	}
	if !allowedImageExt[ext] {
		return "فقط فایل jpg، jpeg، png و webp مجاز است."
	}
	// Double-extension check (image.php.jpg).
	parts := strings.Split(lower, ".")
	if len(parts) > 2 {
		for _, p := range parts[:len(parts)-1] {
			if dangerousExt["."+p] {
				return "نام فایل شامل پسوند غیرمجاز است."
			}
		}
	}
	mime := http.DetectContentType(data)
	if !allowedImageMIME[mime] {
		return "نوع فایل معتبر نیست."
	}
	// Magic bytes must agree with sniffed MIME.
	switch mime {
	case "image/jpeg":
		if len(data) < 3 || data[0] != 0xFF || data[1] != 0xD8 || data[2] != 0xFF {
			return "فایل معتبر نیست."
		}
	case "image/png":
		if len(data) < 8 || !bytes.Equal(data[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
			return "فایل معتبر نیست."
		}
	case "image/webp":
		if len(data) < 12 || !bytes.Equal(data[:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WEBP")) {
			return "فایل معتبر نیست."
		}
	}
	// Decode check for jpeg/png prevents polyglot files; webp decode
	// needs x/image and is covered by magic+sniff above.
	if mime == "image/jpeg" || mime == "image/png" {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return "فایل تصویری قابل خواندن نیست."
		}
		if w, h := int64(cfg.Width), int64(cfg.Height); w <= 0 || h <= 0 || w*h > 25_000_000 {
			return "ابعاد تصویر مجاز نیست."
		}
	}
	// Extension/MIME consistency.
	switch ext {
	case ".jpg", ".jpeg":
		if mime != "image/jpeg" {
			return "پسوند فایل با محتوای آن مطابقت ندارد."
		}
	case ".png":
		if mime != "image/png" {
			return "پسوند فایل با محتوای آن مطابقت ندارد."
		}
	case ".webp":
		if mime != "image/webp" {
			return "پسوند فایل با محتوای آن مطابقت ندارد."
		}
	}
	return ""
}
