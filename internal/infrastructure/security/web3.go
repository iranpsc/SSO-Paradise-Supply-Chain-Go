package security

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"golang.org/x/crypto/sha3"
)

// secp256k1N2 is N/2 for secp256k1, the low-S cutoff. Laravel compares the S
// component against SECP256K1_HALF_N
// ('7FFFFFFF...B20A0') and rejects anything above it (BIP-62 malleability
// rule); the numeric comparison below is equivalent.
var secp256k1N2 = func() *big.Int {
	n, _ := new(big.Int).SetString("7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0", 16)
	return n
}()

// keccak256 returns the legacy Keccak-256 digest Ethereum uses (pre-NIST
// padding, not FIPS-202 SHA3-256).
func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

// ethereumAddress derives the 0x address from a 65-byte uncompressed
// secp256k1 public key (0x04 || X || Y): last 20 bytes of keccak256(X || Y).
func ethereumAddress(uncompressed []byte) string {
	return "0x" + hex.EncodeToString(keccak256(uncompressed[1:])[12:])
}

// personalHash prefixes message exactly like Ethereum personal_sign:
// "\x19Ethereum Signed Message:\n" + byte length + message.
func personalHash(message string) []byte {
	prefix := "\x19Ethereum Signed Message:\n" + itoa(len(message)) + message
	return keccak256([]byte(prefix))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// VerifyWalletSignature reports whether signature is a valid Ethereum
// personal_sign signature of message made by the holder of address. It ports
// Web3AuthController::isValidWalletSignature one rule at a time:
//
//   - signature layout is r (32 bytes) || s (32 bytes) || v (1 byte),
//     0x-prefixed (65 bytes, 130 hex chars);
//   - s must be low (s <= N/2), rejecting malleated counterparts;
//   - v normalizes to 27/28 (values below 27 gain 27) and the recovery id
//     must be 0 or 1;
//   - the address recovered from (r, s, v) must equal address.
//
// Any malformed input returns false, never an error, mirroring the Laravel
// boolean contract.
func VerifyWalletSignature(address, signature, message string) bool {
	if len(signature) != 132 || !strings.HasPrefix(signature, "0x") {
		return false
	}
	rHex, sHex, vHex := signature[2:66], signature[66:130], signature[130:132]
	rBytes, err := hex.DecodeString(rHex)
	if err != nil {
		return false
	}
	sBytes, err := hex.DecodeString(sHex)
	if err != nil {
		return false
	}
	vBytes, err := hex.DecodeString(vHex)
	if err != nil {
		return false
	}
	if new(big.Int).SetBytes(sBytes).Cmp(secp256k1N2) > 0 {
		return false
	}
	v := int(vBytes[0])
	if v < 27 {
		v += 27
	}
	recid := v - 27
	if recid != 0 && recid != 1 {
		return false
	}
	// btcec compact form is header || R || S with header = 27 + recid for
	// uncompressed keys, which is exactly the normalized Ethereum v.
	compact := append([]byte{byte(v)}, append(rBytes, sBytes...)...)
	pub, _, err := ecdsa.RecoverCompact(compact, personalHash(message))
	if err != nil {
		return false
	}
	return strings.EqualFold(ethereumAddress(pub.SerializeUncompressed()), address)
}
