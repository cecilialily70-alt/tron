// Package verify derives TRON addresses from private keys using Go's
// proven crypto libraries (decred secp256k1 + golang.org/x/crypto sha3).
//
// Primary path: CGo wrapper around libsecp256k1 (Bitcoin Core)
// Fallback path: pure Go via decred/secp256k1
package verify

import (
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/sha3"
)

// DeriveHash20 derives the 20-byte TRON raw address from a 32-byte private key.
//
// Automatically selects the fastest available implementation:
//   - libsecp256k1 (Bitcoin Core) via CGo when available (~2-3x faster)
//   - Pure Go decred/secp256k1 as fallback
//
// Steps:
//   1. Private key → secp256k1 public key (X,Y)
//   2. Keccak-256(X || Y)  (64 bytes, big-endian, no 0x04 prefix)
//   3. Take last 20 bytes of the hash
//
// Returns nil if the private key is invalid (zero or >= curve order).
func DeriveHash20(privKeyBytes []byte) []byte {
	if cgoAvailable {
		if result := DeriveHash20CGo(privKeyBytes); result != nil {
			return result
		}
	}
	return deriveHash20Go(privKeyBytes)
}

// deriveHash20Go is the proven pure-Go implementation using decred/secp256k1.
// Used as fallback when libsecp256k1 is not available or CGo derivation fails.
func deriveHash20Go(privKeyBytes []byte) []byte {
	privKey := secp256k1.PrivKeyFromBytes(privKeyBytes)
	if privKey == nil {
		return nil
	}
	pubKey := privKey.PubKey()
	pubBytes := pubKey.SerializeUncompressed() // 65 bytes: 0x04 || X(32) || Y(32)

	k := sha3.NewLegacyKeccak256()
	k.Write(pubBytes[1:]) // skip 0x04 prefix, hash X||Y
	h := k.Sum(nil)       // 32 bytes

	return h[12:32] // last 20 bytes = TRON raw address
}
