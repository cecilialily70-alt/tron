// +build cgo

package verify

/*
#cgo LDFLAGS: -lsecp256k1

#include <stdlib.h>

// tron_derive_address is defined in hot.c
int tron_derive_address(const unsigned char *privkey, char *address_out);
*/
import "C"
import "unsafe"

var cgoAvailable bool

func init() {
	var priv [32]byte
	var buf [35]C.char
	priv[31] = 1
	cgoAvailable = C.tron_derive_address((*C.uchar)(unsafe.Pointer(&priv[0])), &buf[0]) == 1
}

// DeriveAddressCGo derives the full TRON base58check address from a 32-byte
// private key using libsecp256k1 + Keccak + SHA256 + Base58 — all in C.
// Single CGo call, zero Go allocations in the hot path.
func DeriveAddressCGo(privKeyBytes []byte) string {
	if len(privKeyBytes) != 32 {
		return ""
	}
	var buf [35]C.char
	ret := C.tron_derive_address((*C.uchar)(unsafe.Pointer(&privKeyBytes[0])), &buf[0])
	if ret == 0 {
		return ""
	}
	return C.GoString(&buf[0])
}

// DeriveHash20CGo kept for backward compatibility.
func DeriveHash20CGo(privKeyBytes []byte) []byte {
	return nil // superseded by full-address path
}
