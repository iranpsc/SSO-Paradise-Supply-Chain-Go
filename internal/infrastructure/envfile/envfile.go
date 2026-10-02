// Package envfile loads local configuration without replacing deployment/CI
// variables. Secrets are never included in configuration error messages.
package envfile

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func Load() error {
	path := os.Getenv("ENV_FILE")
	explicit := path != ""
	if !explicit {
		path = ".env"
	}
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return nil
	}
	if err != nil {
		return errors.New("environment file is not accessible")
	}
	if err := godotenv.Load(path); err != nil {
		return errors.New("invalid environment file")
	}
	return nil
}

func MustLoad() {
	if err := Load(); err != nil {
		log.Fatal(err)
	}
}
