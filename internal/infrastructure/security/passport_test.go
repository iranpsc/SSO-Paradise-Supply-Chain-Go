package security

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This optional interoperability check uses the actual sibling Passport/League
// implementation. The temporary RSA key and both tokens are synthetic.
func TestJWTInteroperatesWithSiblingLaravelPassport(t *testing.T) {
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("PHP is unavailable")
	}
	autoload, err := filepath.Abs("../../../../SSO-Paradise-Supply-Chain/vendor/autoload.php")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(autoload); err != nil {
		t.Skip("sibling Laravel vendor dependencies are unavailable")
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	j := JWT{key}
	directory := t.TempDir()
	private := filepath.Join(directory, "private.pem")
	public := filepath.Join(directory, "public.pem")
	if err = os.WriteFile(private, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0600); err != nil {
		t.Fatal(err)
	}
	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(public, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes}), 0600); err != nil {
		t.Fatal(err)
	}
	script := `<?php
 require $argv[1];
 $input=json_decode(stream_get_contents(STDIN),true,512,JSON_THROW_ON_ERROR);
 $config=\Lcobucci\JWT\Configuration::forAsymmetricSigner(new \Lcobucci\JWT\Signer\Rsa\Sha256(),\Lcobucci\JWT\Signer\Key\InMemory::file($argv[2]),\Lcobucci\JWT\Signer\Key\InMemory::file($argv[3]));
 $parsed=$config->parser()->parse($input['go_token']);
 $valid=$config->validator()->validate($parsed,new \Lcobucci\JWT\Validation\Constraint\SignedWith($config->signer(),$config->verificationKey()),new \Lcobucci\JWT\Validation\Constraint\PermittedFor('2'),new \Lcobucci\JWT\Validation\Constraint\RelatedTo('3'));
 $client=new \Laravel\Passport\Bridge\Client('2','Interop',['https://client.example/callback'],true,null,['authorization_code']);
 $token=new \Laravel\Passport\Bridge\AccessToken('3',[],$client);
 $token->setIdentifier(str_repeat('b',80));$token->setExpiryDateTime(new \DateTimeImmutable('+1 hour'));
 $token->setPrivateKey(new \League\OAuth2\Server\CryptKey($argv[2],null,false));
 echo json_encode(['go_valid'=>$valid,'passport_token'=>$token->toString()],JSON_THROW_ON_ERROR);
 `
	path := filepath.Join(directory, "interop.php")
	if err = os.WriteFile(path, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	token, err := j.SignAccess(strings.Repeat("a", 80), 2, 3, []string{"profile"}, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(map[string]string{"go_token": token})
	command := exec.Command(php, "-d", "display_errors=stderr", "-d", "log_errors=0", path, autoload, private, public)
	command.Stdin = bytes.NewReader(input)
	var diagnostics bytes.Buffer
	command.Stderr = &diagnostics
	output, err := command.Output()
	if err != nil {
		t.Fatalf("Passport interoperability failed: %v: %s", err, diagnostics.String())
	}
	var result struct {
		Valid bool   `json:"go_valid"`
		Token string `json:"passport_token"`
	}
	if err = json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Passport output is invalid: %v: %s; %s", err, output, diagnostics.String())
	}
	if !result.Valid {
		t.Fatal("Passport rejected Go JWT")
	}
	id, err := j.VerifyAccess(result.Token, time.Now())
	if err != nil || id != strings.Repeat("b", 80) {
		t.Fatalf("Go rejected Passport JWT: %v", err)
	}
}

func TestPassportCipherInteroperatesWithActualDefuse(t *testing.T) {
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("PHP unavailable")
	}
	autoload, _ := filepath.Abs("../../../../SSO-Paradise-Supply-Chain/vendor/autoload.php")
	if _, err = os.Stat(autoload); err != nil {
		t.Skip("Laravel vendor unavailable")
	}
	key := []byte("abcdefghijklmnopqrstuvwxyz012345")
	payload := `{"client_id":"2","refresh_token_id":"` + strings.Repeat("a", 80) + `","expire_time":2000000000}`
	input, _ := json.Marshal(map[string]string{"key": base64.StdEncoding.EncodeToString(key), "payload": payload})
	command := exec.Command(php, "-d", "display_errors=stderr", "-r", `require $argv[1];$input=json_decode(stream_get_contents(STDIN),true);echo \Defuse\Crypto\Crypto::encryptWithPassword($input['payload'],base64_decode($input['key']));`, autoload)
	command.Stdin = bytes.NewReader(input)
	encrypted, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	decoder, err := NewPassportCipher([]byte("base64:" + base64.StdEncoding.EncodeToString(key)))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decoder.Decode(string(encrypted))
	if err != nil || string(decoded) != payload {
		t.Fatal("Go rejected real Passport encrypted payload", err)
	}
	altered := append([]byte(nil), encrypted...)
	if altered[len(altered)-1] == '0' {
		altered[len(altered)-1] = '1'
	} else {
		altered[len(altered)-1] = '0'
	}
	if _, err = decoder.Decode(string(altered)); err == nil {
		t.Fatal("tampered token accepted")
	}
	if _, err = (PassportCipher{Key: bytes.Repeat([]byte{'x'}, 32)}).Decode(string(encrypted)); err == nil {
		t.Fatal("wrong key accepted")
	}
	if _, err = decoder.Decode("00"); err == nil {
		t.Fatal("malformed ciphertext accepted")
	}
}

func TestSignedURLInteroperatesWithSiblingLaravel(t *testing.T) {
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("PHP is unavailable")
	}
	autoload, _ := filepath.Abs("../../../../SSO-Paradise-Supply-Chain/vendor/autoload.php")
	if _, err = os.Stat(autoload); err != nil {
		t.Skip("Laravel vendor unavailable")
	}
	key := "base64:c3ludGhldGljLXRlc3Qta2V5LXN5bnRoZXRpYy1rZXk="
	loaded, err := LoadAppKey(key, "unused")
	if err != nil || string(loaded) != key {
		t.Fatal("Laravel signing key must remain literal")
	}
	url := SignedVerificationURL("https://accounts.example", 3, "test@example.com", time.Now().Add(time.Hour), loaded)
	script := `require $argv[1]; $request=\Illuminate\Http\Request::create($argv[2]); $generator=new \Illuminate\Routing\UrlGenerator(new \Illuminate\Routing\RouteCollection(),$request); $generator->setKeyResolver(fn()=>[$argv[3]]); echo $generator->hasValidSignature($request)?'valid':'invalid';`
	output, err := exec.Command(php, "-d", "display_errors=stderr", "-d", "log_errors=0", "-r", script, autoload, url, key).Output()
	if err != nil || string(output) != "valid" {
		t.Fatalf("Laravel rejected signed verification URL: %v %s", err, output)
	}
}

func TestCookieInteroperatesWithSiblingLaravelEncrypter(t *testing.T) {
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("PHP unavailable")
	}
	autoload, _ := filepath.Abs("../../../../SSO-Paradise-Supply-Chain/vendor/autoload.php")
	if _, err = os.Stat(autoload); err != nil {
		t.Skip("Laravel vendor unavailable")
	}
	key := []byte("abcdefghijklmnopqrstuvwxyz012345")
	name := "laravel_session"
	value := strings.Repeat("s", 40)
	input, _ := json.Marshal(map[string]string{"key": base64.StdEncoding.EncodeToString(key), "name": name, "value": value})
	command := exec.Command(php, "-d", "display_errors=stderr", "-r", `require $argv[1];$x=json_decode(stream_get_contents(STDIN),true);$key=base64_decode($x['key']);$encrypter=new \Illuminate\Encryption\Encrypter($key,'AES-256-CBC');echo $encrypter->encrypt(\Illuminate\Cookie\CookieValuePrefix::create($x['name'],$key).$x['value'],false);`, autoload)
	command.Stdin = bytes.NewReader(input)
	encrypted, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	decoder := PassportCipher{Key: key}
	decoded, err := decoder.DecodeCookie(name, string(encrypted))
	if err != nil || decoded != value {
		t.Fatal("Laravel session cookie rejected", err)
	}
	if decoded, err = decoder.DecodeCookie(name, url.PathEscape(string(encrypted))); err != nil || decoded != value {
		t.Fatal("HTTP-encoded Laravel cookie rejected", err)
	}
	if _, err = decoder.DecodeCookie("other_session", string(encrypted)); err == nil {
		t.Fatal("cookie-name substitution accepted")
	}
	if _, err = (PassportCipher{Key: bytes.Repeat([]byte{'x'}, 32)}).DecodeCookie(name, string(encrypted)); err == nil {
		t.Fatal("wrong cookie key accepted")
	}
	raw, _ := base64.StdEncoding.DecodeString(string(encrypted))
	raw[len(raw)/2] ^= 1
	if _, err = decoder.DecodeCookie(name, base64.StdEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("tampered cookie accepted")
	}
}
