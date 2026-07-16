package checker

import (
	"strings"

	"tron-address-generator/verify"
)

type MatchType int

const (
	Suffix7 MatchType = iota
	SixSixes
	SixEights
)

type Match struct {
	Address    string
	PrivateKey string
	Pattern    byte
	Type       MatchType
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

func Check(privateKey []byte) *Match {
	addr := verify.DeriveAddress(privateKey)
	if addr == "" {
		return nil
	}

	if c, ok := checkLastN(addr, 7); ok {
		return &Match{Address: addr, PrivateKey: fmtHex(privateKey), Pattern: c, Type: Suffix7}
	}
	if strings.HasSuffix(addr, "666666") {
		return &Match{Address: addr, PrivateKey: fmtHex(privateKey), Pattern: '6', Type: SixSixes}
	}
	if strings.HasSuffix(addr, "888888") {
		return &Match{Address: addr, PrivateKey: fmtHex(privateKey), Pattern: '8', Type: SixEights}
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
