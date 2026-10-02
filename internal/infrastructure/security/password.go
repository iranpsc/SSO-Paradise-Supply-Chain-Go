package security

import "golang.org/x/crypto/bcrypt"

type Bcrypt struct{ Cost int }

func (b Bcrypt) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.Cost)
	return string(hash), err
}
func (b Bcrypt) Matches(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
