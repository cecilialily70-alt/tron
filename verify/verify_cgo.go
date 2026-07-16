// Package verify - libsecp256k1 (Bitcoin Core) CGo accelerated path.
// Requires: sudo apt install libsecp256k1-dev
// +build cgo

package verify

/*
#cgo LDFLAGS: -lsecp256k1
#include <secp256k1.h>
#include <stdlib.h>

static secp256k1_context* sctx = NULL;

static secp256k1_context* tron_init(void) {
    if (sctx == NULL) {
        sctx = secp256k1_context_create(SECP256K1_CONTEXT_SIGN);
    }
    return sctx;
}

static int tron_ec_pubkey(const unsigned char *priv, unsigned char *pub) {
    secp256k1_context *ctx = tron_init();
    if (!ctx) return 0;
    secp256k1_pubkey pk;
    if (!secp256k1_ec_pubkey_create(ctx, &pk, priv)) return 0;
    size_t len = 65;
    secp256k1_ec_pubkey_serialize(ctx, pub, &len, &pk, SECP256K1_EC_UNCOMPRESSED);
    return 1;
}
*/
import "C"
import (
	"unsafe"

	"golang.org/x/crypto/sha3"
)

var cgoAvailable bool

func init() {
	var priv [32]byte
	var pub [65]C.uchar
	priv[31] = 1
	cgoAvailable = C.tron_ec_pubkey((*C.uchar)(unsafe.Pointer(&priv[0])), &pub[0]) == 1
}

// DeriveHash20CGo uses libsecp256k1 (Bitcoin Core library) for pubkey derivation.
func DeriveHash20CGo(privKeyBytes []byte) []byte {
	if len(privKeyBytes) != 32 {
		return nil
	}
	var pubkey [65]C.uchar
	ret := C.tron_ec_pubkey((*C.uchar)(unsafe.Pointer(&privKeyBytes[0])), &pubkey[0])
	if ret == 0 {
		return nil
	}
	k := sha3.NewLegacyKeccak256()
	k.Write((*[64]byte)(unsafe.Pointer(&pubkey[1]))[:])
	h := k.Sum(nil)
	return h[12:32]
}
