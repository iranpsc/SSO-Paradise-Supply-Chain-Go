package security_test

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"golang.org/x/crypto/sha3"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/infrastructure/security"
)

var (
	secpN = btcec.S256().N
	halfN = new(big.Int).Rsh(new(big.Int).Set(secpN), 1)
)

func keccak(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

// ethAddress derives the 0x address of a test key independently of the
// package under test.
func ethAddress(priv *btcec.PrivateKey) string {
	pub := priv.PubKey().SerializeUncompressed()
	return "0x" + hex.EncodeToString(keccak(pub[1:])[12:])
}

func privKey(t *testing.T, hexSeed string) *btcec.PrivateKey {
	t.Helper()
	seed := make([]byte, 32)
	if hexSeed != "" {
		b, err := hex.DecodeString(hexSeed)
		if err != nil {
			t.Fatal(err)
		}
		copy(seed[32-len(b):], b)
	} else {
		if _, err := rand.Read(seed); err != nil {
			t.Fatal(err)
		}
	}
	priv, _ := btcec.PrivKeyFromBytes(seed)
	return priv
}

// signPersonal produces an Ethereum personal_sign signature (r || s || v,
// 0x-prefixed) using btcec, normalizing to low-S exactly like honest
// wallets do.
func signPersonal(t *testing.T, priv *btcec.PrivateKey, message string) string {
	t.Helper()
	hash := keccak([]byte("\x19Ethereum Signed Message:\n" + strconv.Itoa(len(message)) + message))
	compact := ecdsa.SignCompact(priv, hash, false)
	r, sBytes, v := compact[1:33], compact[33:65], compact[0]
	if new(big.Int).SetBytes(sBytes).Cmp(halfN) > 0 {
		sBytes = pad32(new(big.Int).Sub(secpN, new(big.Int).SetBytes(sBytes)).Bytes())
		v = 27 + ((v - 27) ^ 1)
	}
	return "0x" + hex.EncodeToString(append(append(r, sBytes...), v))
}

func pad32(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func TestVerifyWalletSignatureRoundTrip(t *testing.T) {
	priv := privKey(t, "")
	address := ethAddress(priv)
	message := "Sign in to Laravel at localhost.\n\nWallet: " + strings.ToLower(address) + "\nNonce: abc123XYZ456def789GHI012jkl345mn"
	if !security.VerifyWalletSignature(address, signPersonal(t, priv, message), message) {
		t.Fatal("valid personal_sign signature rejected")
	}
}

func TestVerifyWalletSignatureKnownVector(t *testing.T) {
	// Private key 0x01 belongs to 0x7e5f4552091a69125d5dfcb7b8c2659029395bdf
	// on every Ethereum implementation; the address is hardcoded so key
	// recovery and Keccak derivation are checked against external truth.
	priv := privKey(t, "01")
	const address = "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf"
	if got := ethAddress(priv); !strings.EqualFold(got, address) {
		t.Fatalf("address derivation mismatch: %s", got)
	}
	message := "hello wallet"
	sig := signPersonal(t, priv, message)
	if !security.VerifyWalletSignature(address, sig, message) {
		t.Fatal("known-vector signature rejected")
	}
	// EIP-55 style checksummed casing must also verify.
	if !security.VerifyWalletSignature("0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf", sig, message) {
		t.Fatal("checksummed address rejected")
	}
}

// TestVerifyWalletSignaturePHPSignedVectors verifies signatures produced by
// the exact libraries Laravel uses (Elliptic\EC + kornrunner\Keccak, same
// code as Web3AuthTest::signMessage), so the Go verifier is checked against
// non-Go cryptography. Vectors were generated once and are fixed.
func TestVerifyWalletSignaturePHPSignedVectors(t *testing.T) {
	vectors := []struct{ address, message, signature string }{
		{
			"0x7e5f4552091a69125d5dfcb7b8c2659029395bdf",
			"Sign in to Laravel at localhost.\n\nWallet: 0x7e5f4552091a69125d5dfcb7b8c2659029395bdf\nNonce: fixedVectorOne00000000000000000001",
			"0x7059b7ba5f444b422fb164db7bcbcba252075cf94edf6dbebffdd5f96c411e9d4796f8c54fcb31899aab072086701c54c3606bf17357605e442a65b736c872fa1b",
		},
		{
			"0xdefa9efd2d940717f87a65373b8f9a52f9b48786",
			"Link wallet to your Laravel account at localhost.\n\nAccount ID: 42\nWallet: 0xdefa9efd2d940717f87a65373b8f9a52f9b48786\nNonce: fixedVectorTwo00000000000000000002",
			"0x2ea2079cb493d307ba432cc368747438a25fe712afd4f37575f82bb8774a58036551c6c2fcc18e4a5283247b6b59022735bcc1ccfdf76d65f528c2373f239f4a1b",
		},
	}
	for i, v := range vectors {
		if !security.VerifyWalletSignature(v.address, v.signature, v.message) {
			t.Errorf("vector %d: PHP-signed signature rejected", i)
		}
		if security.VerifyWalletSignature(v.address, v.signature, v.message+"tampered") {
			t.Errorf("vector %d: tampered message accepted", i)
		}
	}
}

func TestVerifyWalletSignatureRejects(t *testing.T) {
	priv := privKey(t, "")
	address := ethAddress(priv)
	message := "Link wallet to your Laravel account at localhost.\n\nAccount ID: 7\nWallet: " + address + "\nNonce: xyz"
	valid := signPersonal(t, priv, message)

	flip := func(s string, pos int) string {
		b, _ := hex.DecodeString(s[2:])
		b[pos] ^= 0x01
		return "0x" + hex.EncodeToString(b)
	}
	highS := func(s string) string {
		b, _ := hex.DecodeString(s[2:])
		r, sv, v := b[:32], b[32:64], b[64]
		neg := pad32(new(big.Int).Sub(secpN, new(big.Int).SetBytes(sv)).Bytes())
		return "0x" + hex.EncodeToString(append(append(r, neg...), (v-27)^1+27))
	}

	cases := map[string]struct {
		address   string
		signature string
		message   string
	}{
		"wrong address":      {flip(address, 19), valid, message},
		"tampered message":   {address, valid, message + " (edited)"},
		"flipped r bit":      {address, flip(valid, 0), message},
		"flipped s bit":      {address, flip(valid, 40), message},
		"high s is rejected": {address, highS(valid), message},
		"v out of range":     {address, valid[:130] + "1e", message},
		"too short":          {address, valid[:100], message},
		"missing prefix":     {address, valid[2:], message},
		"non hex":            {address, valid[:10] + "zz" + valid[12:], message},
		"empty":              {address, "", message},
	}
	for name, c := range cases {
		if security.VerifyWalletSignature(c.address, c.signature, c.message) {
			t.Errorf("%s: invalid signature accepted", name)
		}
	}
}

func TestVerifyWalletSignatureAcceptsRawRecoveryID(t *testing.T) {
	// Some signers emit v as 0/1; Laravel normalizes with v += 27.
	priv := privKey(t, "")
	address := ethAddress(priv)
	message := "raw v test"
	sig := signPersonal(t, priv, message)
	var rawV string
	switch v := sig[130:132]; v {
	case "1b":
		rawV = "00"
	case "1c":
		rawV = "01"
	default:
		t.Fatalf("unexpected v %s", v)
	}
	if !security.VerifyWalletSignature(address, sig[:130]+rawV, message) {
		t.Fatal("raw recovery id signature rejected")
	}
}
