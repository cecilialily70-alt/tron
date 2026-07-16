//go:build !cgo

package verify

var cgoAvailable = false

func DeriveHash20CGo(privKeyBytes []byte) []byte {
	return nil
}
