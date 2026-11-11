// Regression test for the GHSA-4w3g-jfj2-cw2q backport described in
// ../FACET-PATCH.md.

package josecipher

import (
	"crypto/aes"
	"testing"
)

func TestOpenEmptyCiphertextErrors(t *testing.T) {
	aead, err := NewCBCHMAC(make([]byte, 32), aes.NewCipher)
	if err != nil {
		t.Fatal(err)
	}
	nonce, aad := make([]byte, 16), []byte("aad")
	// A valid tag over an empty ciphertext reaches unpadBuffer, which used to index an empty slice.
	tag := aead.(*cbcAEAD).computeAuthTag(aad, nonce, nil)
	if _, err := aead.Open(nil, nonce, tag, aad); err == nil {
		t.Fatal("empty ciphertext must be rejected")
	}
}
