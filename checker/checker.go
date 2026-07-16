// Package checker validates TRON vanity address patterns.
//
// Uses Go's trusted secp256k1 + Keccak256 to derive addresses from private keys,
// then checks for:
//   - 7-character identical prefix/suffix
//   - 6 consecutive "6"s or "8"s anywhere in the address
package checker

import (
	"crypto/sha256"
	"strings"

	"github.com/mr-tron/base58"
	"tron-address-generator/verify"
)

type MatchType int

const (
	Suffix7   MatchType = iota // last 7 chars identical
	Prefix7                    // first 7 chars identical (after T)
	SixSixes                  // six consecutive 6s
	SixEights                 // six consecutive 8s
)

type Match struct {
	Address    string
	PrivateKey string
	Pattern    byte
	Type       MatchType
}

func buildPayload(hash20 []byte) []byte {
	payload := make([]byte, 25)
	payload[0] = 0x41
	copy(payload[1:21], hash20)
	h1 := sha256.Sum256(payload[:21])
	h2 := sha256.Sum256(h1[:])
	copy(payload[21:25], h2[:4])
	return payload
}

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

func checkSixConsecutive(address string, target byte) bool {
	return strings.Contains(address, strings.Repeat(string(target), 6))
}

// Check does the full address derivation and vanity pattern check.
func Check(privateKey []byte) *Match {
	hash20 := verify.DeriveHash20(privateKey)
	if hash20 == nil {
		return nil
	}

	payload := buildPayload(hash20)
	address := base58.Encode(payload)

	// Priority 1: 7-char identical prefix/suffix
	if c, ok := checkLastN(address, 7); ok {
		return &Match{Address: address, PrivateKey: fmtHex(privateKey), Pattern: c, Type: Suffix7}
	}
	if c, ok := checkFirstN(address, 7); ok {
		return &Match{Address: address, PrivateKey: fmtHex(privateKey), Pattern: c, Type: Prefix7}
	}

	// Priority 2: 6 consecutive 6s or 8s anywhere
	if checkSixConsecutive(address, '6') {
		return &Match{Address: address, PrivateKey: fmtHex(privateKey), Pattern: '6', Type: SixSixes}
	}
	if checkSixConsecutive(address, '8') {
		return &Match{Address: address, PrivateKey: fmtHex(privateKey), Pattern: '8', Type: SixEights}
	}
	return nil
}

func fmtHex(data []byte) string {
	const hexChars = "0123456789abcdef"
	out := make([]byte, len(data)*2)
	for i, b := range data {
		out[i*2] = hexChars[b>>4]
		out[i*2+1] = hexChars[b&0x0F]
	}
	return string(out)
}
