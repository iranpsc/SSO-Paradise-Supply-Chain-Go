package domain

import "regexp"

// Ethereum address and personal_sign signature shapes. They mirror the
// Laravel Web3AuthController validation rules exactly:
//
//	'address'   => 'regex:/^0x[a-fA-F0-9]{40}$/'
//	'signature' => 'regex:/^0x[a-fA-F0-9]{130}$/'
var (
	walletAddressPattern   = regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)
	walletSignaturePattern = regexp.MustCompile(`^0x[0-9a-fA-F]{130}$`)
)

// ValidWalletAddress reports whether s is a 20-byte hex Ethereum address.
func ValidWalletAddress(s string) bool { return walletAddressPattern.MatchString(s) }

// ValidWalletSignature reports whether s is a 65-byte hex Ethereum signature.
func ValidWalletSignature(s string) bool { return walletSignaturePattern.MatchString(s) }

// Wallet link outcomes, mirroring Web3AuthController::attachWallet which
// returns 'success', 'already_connected' or 'already_linked'.
const (
	WalletLinkSuccess          = "success"
	WalletLinkAlreadyConnected = "already_connected"
	WalletLinkAlreadyLinked    = "already_linked"
)
