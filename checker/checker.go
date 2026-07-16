// Package checker validates TRON vanity address patterns and verifies
// that generated private keys correctly correspond to their addresses.
//
// Always performs full base58 encoding for every candidate, then checks both
// prefix and suffix 3-char patterns. All matches are verified by re-deriving
// the address from the private key using Go's trusted crypto libraries.
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

// CheckFull validates a potential vanity address candidate.
//
// Process:
//  1. Build base58check payload from GPU-derived hash20
//  2. Full base58 encoding of the payload
//  3. CRITICAL: re-derive hash20 from private key using trusted Go crypto
//     and compare — rejects any key-address mismatch
//  4. Check suffix 3-char pattern
//  5. Check prefix 3-char pattern
//  6. Return match only if verification passes AND a pattern is found
func CheckFull(privateKey, hash20 []byte) *Match {
	payload := buildPayload(hash20)
	address := base58.Encode(payload)

	// === CRITICAL VERIFICATION ===
	// Re-derive the address from the private key using Go's trusted
	// secp256k1 + Keccak256 implementation. This guarantees the private
	// key actually controls this address on the TRON blockchain.
	derivedHash20 := verify.DeriveHash20(privateKey)
	if derivedHash20 == nil {
		return nil
	}
	if !bytes.Equal(hash20, derivedHash20) {
		return nil
	}

	// === Pattern matching ===
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
