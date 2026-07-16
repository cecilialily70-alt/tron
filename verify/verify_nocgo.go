//go:build !cgo

package verify

var cgoAvailable = false

func DeriveAddressCGo(privKeyBytes []byte) string {
	return ""
}

func DeriveHash20CGo(privKeyBytes []byte) []byte {
	return nil
}
