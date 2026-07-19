package checker

import (
	"tron-address-generator/verify"
)

type Match struct {
	Address    string
	PrivateKey string
	Pattern    byte // the repeated character
}

// checkLastN verifies that the last N characters of a string are all the same.
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

// Check returns a Match if the private key yields an address whose last 6
// characters are all identical (e.g. ...AAAAAA, ...111111, ...666666).
func Check(privateKey []byte) *Match {
	addr := verify.DeriveAddress(privateKey)
	if addr == "" {
		return nil
	}

	if c, ok := checkLastN(addr, 6); ok {
		return &Match{Address: addr, PrivateKey: fmtHex(privateKey), Pattern: c}
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
