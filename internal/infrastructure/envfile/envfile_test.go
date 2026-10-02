package envfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileLoadingPreservesDeploymentVariablesAndInterpolatesNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("PARADISE_TEST_DEPLOYMENT=file\nPARADISE_TEST_NAME='IRPSC'\nPARADISE_TEST_FROM=\"${PARADISE_TEST_NAME}\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENV_FILE", path)
	t.Setenv("PARADISE_TEST_DEPLOYMENT", "ci")
	for _, key := range []string{"PARADISE_TEST_NAME", "PARADISE_TEST_FROM"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("PARADISE_TEST_DEPLOYMENT") != "ci" || os.Getenv("PARADISE_TEST_FROM") != "IRPSC" {
		t.Fatal("file replaced deployment configuration or failed interpolation")
	}
}

func TestExplicitMissingAndMalformedFilesFailWithoutSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	t.Setenv("ENV_FILE", path)
	if err := Load(); err == nil {
		t.Fatal("explicit missing file was ignored")
	}
	if err := os.WriteFile(path, []byte("BAD='private-credential\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Load(); err == nil || strings.Contains(err.Error(), "private-credential") {
		t.Fatal("invalid file accepted or secret leaked")
	}
}
