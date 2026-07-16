// Package checker validates TRON vanity address patterns.
//
// Uses Go's trusted secp256k1 + Keccak256 to derive addresses from private keys,
// then checks for 5-character prefix/suffix vanity patterns.
package checker

import (
	"crypto/sha256"

	"github.com/mr-tron/base58"
	"tron-address-generator/verify"
)

type MatchType int

const (
	Suffix5 MatchType = iota
	Prefix5
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

func Check(privateKey []byte) *Match {
	hash20 := verify.DeriveHash20(privateKey)
	if hash20 == nil {
		return nil
	}

	payload := buildPayload(hash20)
	address := base58.Encode(payload)

	if c, ok := checkLastN(address, 5); ok {
		return &Match{Address: address, PrivateKey: fmtHex(privateKey), Pattern: c, Type: Suffix5}
	}
	if c, ok := checkFirstN(address, 5); ok {
		return &Match{Address: address, PrivateKey: fmtHex(privateKey), Pattern: c, Type: Prefix5}
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
