package domain

import "testing"

func TestPasswordPolicy(t *testing.T) {
	for _, p := range []string{"short", "abcdefgh1!", "ABCDEFGH1!", "Abcdefghi!", "Abcdefghi1", "Ab1!", "Ab1!aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {
		if len(ValidatePassword(p, p)) == 0 {
			t.Errorf("accepted weak password %q", p)
		}
	}
	if len(ValidatePassword("Secure!2026", "Secure!2026")) != 0 {
		t.Fatal("rejected valid password")
	}
	if len(ValidatePassword("Secure!2026", "Other!2026")) == 0 {
		t.Fatal("accepted mismatched confirmation")
	}
}
func TestEmailValidationRejectsDisplayNameAndInjection(t *testing.T) {
	for _, email := range []string{"", "not-an-email", "Name <a@example.com>", "a@example.com\r\nBcc: attacker@example.com"} {
		if ValidateIdentity("Name", email, 50)["email"] == "" {
			t.Errorf("accepted %q", email)
		}
	}
}
