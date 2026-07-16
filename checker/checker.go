// Package checker validates TRON vanity address patterns and verifies
// that generated private keys correctly correspond to their addresses.
//
// Hot path: base58 encode every key, check prefix+suffix patterns.
// Verification (secp256k1 re-derivation) runs ONLY on pattern matches (~1/1682 keys),
// keeping throughput at millions per second.
package checker

import (
	"bytes"
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

// verifyMatch re-derives the hash20 from the private key using Go's trusted
// secp256k1 + Keccak256 and compares with the GPU-derived hash20.
// Returns true only if the private key correctly controls this address.
//
// This is EXPENSIVE (~100-500us per call). Only call on pattern matches.
func verifyMatch(privateKey, hash20 []byte) bool {
	derived := verify.DeriveHash20(privateKey)
	if derived == nil {
		return false
	}
	return bytes.Equal(hash20, derived)
}

// CheckFull validates a potential vanity address candidate.
//
// Hot path (every key, must be fast):
//  1. Build base58check payload (SHA256x2, <1us)
//  2. Base58 encode (big.Int, ~5-10us)
//  3. Check suffix + prefix patterns (string compare, nanoseconds)
//
// Verification path (only on pattern matches, ~1/1682 keys):
//  4. Re-derive hash20 from private key using trusted Go crypto (~100us)
//  5. Reject if mismatch (GPU computation error)
func CheckFull(privateKey, hash20 []byte) *Match {
	payload := buildPayload(hash20)
	address := base58.Encode(payload)

	// Fast pattern check — no secp256k1 in this path
	var matchType MatchType
	var pattern byte
	if c, ok := checkSuffix3(address); ok {
		matchType = Suffix3
		pattern = c
	} else if c, ok := checkPrefix3(address); ok {
		matchType = Prefix3
		pattern = c
	} else {
		return nil
	}

	// Verification ONLY on pattern match candidates
	if !verifyMatch(privateKey, hash20) {
		return nil
	}

	return &Match{
		Address:    address,
		PrivateKey: fmtHex(privateKey),
		Pattern:    pattern,
		Type:       matchType,
	}
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
