// Package checker validates TRON vanity address patterns.
//
// Uses Go's trusted secp256k1 + Keccak256 to derive addresses from private keys,
// then checks for 6-character prefix/suffix vanity patterns.
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
	Suffix6 MatchType = iota
	Prefix6
)

// Match holds a found vanity address along with its private key and metadata.
type Match struct {
	Address    string
	PrivateKey string
	Pattern    byte
	Type       MatchType
	VanityLen  int
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

// checkLastN verifies that the last N characters of the address are identical.
func checkLastN(address string, n int) (byte, bool) {
	if len(address) < n+1 {
		return 0, false
	}
	c := address[len(address)-1]
	for i := 1; i < n; i++ {
		if address[len(address)-1-i] != c {
			return 0, false
		}
	}
	return c, true
}

// checkFirstN verifies that the first N characters after the leading 'T'
// are identical (e.g., TAAAAA..., TNNNNN...).
func checkFirstN(address string, n int) (byte, bool) {
	if len(address) < n+1 {
		return 0, false
	}
	c := address[1]
	for i := 1; i < n; i++ {
		if address[1+i] != c {
			return 0, false
		}
	}
	return c, true
}

// Check does the full address derivation and vanity pattern check.
// All computation uses Go's trusted crypto libraries — secp256k1, Keccak256, SHA256.
//
// Returns nil if the private key is invalid (zero or >= curve order)
// or if no 6-char prefix/suffix pattern is found.
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

	// Step 4: Pattern matching — 6-char prefix or suffix
	if c, ok := checkLastN(address, 6); ok {
		return &Match{Address: address, PrivateKey: fmtHex(privateKey), Pattern: c, Type: Suffix6, VanityLen: 6}
	}
	if c, ok := checkFirstN(address, 6); ok {
		return &Match{Address: address, PrivateKey: fmtHex(privateKey), Pattern: c, Type: Prefix6, VanityLen: 6}
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
