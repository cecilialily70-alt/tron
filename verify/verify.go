// Package verify derives TRON addresses from private keys.
//
// Primary path (CGO enabled): C function tron_derive_address() — libsecp256k1
//   + Keccak-256 + SHA-256 + Base58, all compiled to native code, single call.
//
// Fallback path (pure Go): Secure but slower, only used when CGo is unavailable.
package verify

import (
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/mr-tron/base58"
	"crypto/sha256"
	"golang.org/x/crypto/sha3"
)

// DeriveAddress derives the full TRON base58check address from a 32-byte
// private key. Uses the CGo fast path when available.
// Returns "" if the private key is invalid.
func DeriveAddress(privKeyBytes []byte) string {
	if cgoAvailable {
		if addr := DeriveAddressCGo(privKeyBytes); addr != "" {
			return addr
		}
	}
	return deriveAddressGo(privKeyBytes)
}

// DeriveHash20 kept for backward compatibility with tests.
func DeriveHash20(privKeyBytes []byte) []byte {
	return deriveHash20Go(privKeyBytes)
}

// deriveAddressGo — pure Go fallback (used only when CGo is unavailable).
func deriveAddressGo(privKeyBytes []byte) string {
	h := deriveHash20Go(privKeyBytes)
	if h == nil {
		return ""
	}
	payload := make([]byte, 25)
	payload[0] = 0x41
	copy(payload[1:21], h)
	s1 := sha256.Sum256(payload[:21])
	s2 := sha256.Sum256(s1[:])
	copy(payload[21:25], s2[:4])
	return base58.Encode(payload)
}

// deriveHash20Go — pure Go hash20 derivation (used as DeriveHash20 fallback).
func deriveHash20Go(privKeyBytes []byte) []byte {
	privKey := secp256k1.PrivKeyFromBytes(privKeyBytes)
	if privKey == nil {
		return nil
	}
	pubKey := privKey.PubKey()
	pubBytes := pubKey.SerializeUncompressed()
	k := sha3.NewLegacyKeccak256()
	k.Write(pubBytes[1:])
	return k.Sum(nil)[12:32]
}
