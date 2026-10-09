package security

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEncryptedFileSessionInteroperatesWithLaravel(t *testing.T) {
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("PHP unavailable")
	}
	autoload, _ := filepath.Abs("../../../../SSO-Paradise-Supply-Chain/vendor/autoload.php")
	if _, err := os.Stat(autoload); err != nil {
		t.Skip("Laravel vendor unavailable")
	}
	key := []byte("abcdefghijklmnopqrstuvwxyz012345")
	input, _ := json.Marshal(map[string]string{"key": base64.StdEncoding.EncodeToString(key)})
	cmd := exec.Command(php, "-d", "display_errors=stderr", "-r", `require $argv[1];$x=json_decode(stream_get_contents(STDIN),true);$e=new \Illuminate\Encryption\Encrypter(base64_decode($x['key']),'AES-256-CBC');echo $e->encrypt(serialize(['login_web_test'=>7,'wallet_login'=>true,'_token'=>'csrf']));`, autoload)
	cmd.Stdin = bytes.NewReader(input)
	encrypted, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	session, err := ParseLaravelFileSession(encrypted, []byte("base64:"+base64.StdEncoding.EncodeToString(key)))
	if err != nil || session["login_web_test"] != int64(7) || session["wallet_login"] != true || session["_token"] != "csrf" {
		t.Fatal("encrypted session rejected", session, err)
	}
	if _, err := ParseLaravelFileSession(encrypted, bytes.Repeat([]byte{'x'}, 32)); err == nil {
		t.Fatal("wrong key accepted")
	}
	encrypted[len(encrypted)/2] ^= 1
	if _, err := ParseLaravelFileSession(encrypted, key); err == nil {
		t.Fatal("tampered session accepted")
	}
}
