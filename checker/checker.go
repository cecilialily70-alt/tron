// Package checker validates TRON vanity address patterns.
//
// Uses Go's trusted secp256k1 + Keccak256 to derive addresses from private keys,
// then checks for 3-character prefix/suffix vanity patterns.
// No untrusted computation — every step is verified Go crypto.
package checker

import (
	"crypto/sha256"

	"github.com/mr-tron/base58"
	"tron-address-generator/verify"
)

// MatchType indicates whether a vanity match is a prefix or suffix pattern.
type MatchType int

const (
	Suffix3 MatchType = iota
	Prefix3
)

// Match holds a found vanity address along with its private key and metadata.
type Match struct {
	Address    string
	PrivateKey string
	Pattern    byte
	Type       MatchType
}

// buildPayload constructs a 25-byte TRON base58check payload:
//
//	[0x41] + [20-byte address hash] + [4-byte double-SHA256 checksum]
func buildPayload(hash20 []byte) []byte {
	payload := make([]byte, 25)
	payload[0] = 0x41
	copy(payload[1:21], hash20)
	h1 := sha256.Sum256(payload[:21])
	h2 := sha256.Sum256(h1[:])
	copy(payload[21:25], h2[:4])
	return payload
}

// checkSuffix3 verifies that the last 3 characters of the address are identical.
func checkSuffix3(address string) (byte, bool) {
	if len(address) < 4 {
		return 0, false
	}
	c := address[len(address)-1]
	if address[len(address)-2] != c || address[len(address)-3] != c {
		return 0, false
	}
	return c, true
}

// checkPrefix3 verifies that the first 3 characters after the leading 'T'
// are identical (e.g., TAAA..., TNNN...).
func checkPrefix3(address string) (byte, bool) {
	if len(address) < 4 {
		return 0, false
	}
	c := address[1]
	if address[2] != c || address[3] != c {
		return 0, false
	}
	return c, true
}

// Check does the full address derivation and vanity pattern check.
// All computation uses Go's trusted crypto libraries — secp256k1, Keccak256, SHA256.
//
// Returns nil if the private key is invalid (zero or >= curve order)
// or if no 3-char prefix/suffix pattern is found.
func Check(privateKey []byte) *Match {
	// Step 1: Trusted secp256k1 derivation (decred library)
	hash20 := verify.DeriveHash20(privateKey)
	if hash20 == nil {
		return nil
	}

	// Step 2: Build base58check payload
	payload := buildPayload(hash20)

	// Step 3: Encode to TRON base58check address
	address := base58.Encode(payload)

	// Step 4: Pattern matching (both prefix and suffix)
	if c, ok := checkSuffix3(address); ok {
		return &Match{Address: address, PrivateKey: fmtHex(privateKey), Pattern: c, Type: Suffix3}
	}
	if c, ok := checkPrefix3(address); ok {
		return &Match{Address: address, PrivateKey: fmtHex(privateKey), Pattern: c, Type: Prefix3}
	}
	return nil
}

// fmtHex converts a byte slice to a lowercase hex string.
func fmtHex(data []byte) string {
	const hexChars = "0123456789abcdef"
	out := make([]byte, len(data)*2)
	for i, b := range data {
		out[i*2] = hexChars[b>>4]
		out[i*2+1] = hexChars[b&0x0F]
	}
	return string(out)
}
