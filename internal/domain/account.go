package domain

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("email already registered")
	ErrUsernameConflict = errors.New("username already registered")
	ErrCredentials      = errors.New("invalid credentials")
	ErrUnverified       = errors.New("email verification required")
	ErrToken            = errors.New("invalid or expired token")
)

type Validation map[string]string

func (v Validation) Error() string { return "validation failed" }

type User struct {
	Mobile       *string   `json:"mobile"`
	UpdatedAt    time.Time `json:"updated_at"`
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Code         *string   `json:"code"`
	// Wallet holds the linked Ethereum address. It lives in the separate
	// wallets table (one address per user) and is nil when unlinked, matching
	// Laravel where wallet_address is a nullable, non-hidden attribute.
	Wallet          *string    `json:"wallet_address"`
	Referral        string     `json:"referral,omitempty"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,29}$`)

func ValidUsername(value string) bool { return usernamePattern.MatchString(value) }

func ValidateIdentity(name, email string, maxName int) Validation {
	v := Validation{}
	if n := utf8.RuneCountInString(name); n == 0 || n > maxName || strings.HasPrefix(strings.ToLower(name), "hm-") {
		v["name"] = "نام معتبر وارد کنید؛ پیشوند رزروشدهٔ حساب‌های کیف پول مجاز نیست."
	}
	a, err := mail.ParseAddress(email)
	if err != nil || a.Address != email || len(email) > 255 {
		v["email"] = "ایمیل معتبر وارد کنید."
	}
	return v
}

func ValidatePassword(password, confirmation string) Validation {
	v := Validation{}
	var upper, lower, digit, symbol bool
	for _, r := range password {
		upper = upper || unicode.IsUpper(r)
		lower = lower || unicode.IsLower(r)
		digit = digit || unicode.IsDigit(r)
		symbol = symbol || unicode.IsPunct(r) || unicode.IsSymbol(r)
	}
	if n := utf8.RuneCountInString(password); n < 8 || n > 40 || len(password) > 72 || !upper || !lower || !digit || !symbol {
		v["password"] = "رمز باید ۸ تا ۴۰ نویسه و شامل حروف بزرگ و کوچک، عدد و نماد باشد."
	}
	if password != confirmation {
		v["password_confirmation"] = "تکرار رمز مطابقت ندارد."
	}
	return v
}

// Laravel reset uses min:8 and confirmation. Change-password adds complexity;
// both retain bcrypt's 72-byte limit to avoid silently truncated passwords.
func ValidateResetPassword(password, confirmation string) Validation {
	v := Validation{}
	if utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		v["password"] = "رمز باید حداقل ۸ نویسه و حداکثر ۷۲ بایت باشد."
	}
	if password != confirmation {
		v["password_confirmation"] = "تکرار رمز مطابقت ندارد."
	}
	return v
}
func ValidateChangedPassword(password, confirmation string) Validation {
	v := ValidatePassword(password, confirmation)
	if utf8.RuneCountInString(password) > 40 && len(password) <= 72 {
		var upper, lower, digit, symbol bool
		for _, r := range password {
			upper = upper || unicode.IsUpper(r)
			lower = lower || unicode.IsLower(r)
			digit = digit || unicode.IsDigit(r)
			symbol = symbol || unicode.IsPunct(r) || unicode.IsSymbol(r)
		}
		if upper && lower && digit && symbol {
			delete(v, "password")
		}
	}
	return v
}
