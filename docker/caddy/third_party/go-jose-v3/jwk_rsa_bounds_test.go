// Regression tests for the GHSA-mgcc-cww5-fp2q backport described in
// FACET-PATCH.md. Upstream v3.0.5 test files are not included in this copy.

package jose

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func b64(n *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(n.Bytes())
}

// oddWithBits returns an odd number with exactly the given bit length. It is
// not prime, so rsa.PrivateKey.Validate would reject it eventually, but only
// after doing work that grows with the size of the values.
func oddWithBits(bits int) *big.Int {
	n := new(big.Int).Lsh(big.NewInt(1), uint(bits-1))
	return n.Or(n, big.NewInt(1))
}

func publicJWK(n *big.Int) []byte {
	return []byte(fmt.Sprintf(`{"kty":"RSA","n":%q,"e":"AQAB"}`, b64(n)))
}

func privateJWK(n, d, p, q *big.Int) []byte {
	return []byte(fmt.Sprintf(`{"kty":"RSA","n":%q,"e":"AQAB","d":%q,"p":%q,"q":%q}`,
		b64(n), b64(d), b64(p), b64(q)))
}

func TestRSAPublicJWKModulusBound(t *testing.T) {
	var key JSONWebKey
	if err := json.Unmarshal(publicJWK(oddWithBits(MaxRSAModulusBits)), &key); err != nil {
		t.Fatalf("modulus at the limit rejected: %v", err)
	}

	err := json.Unmarshal(publicJWK(oddWithBits(MaxRSAModulusBits+1)), &key)
	if err == nil || !strings.Contains(err.Error(), "modulus too large") {
		t.Fatalf("oversized public modulus: got %v, want modulus too large", err)
	}
}

func TestRSAPrivateJWKRejectedBeforeValidate(t *testing.T) {
	// n = p*q and d < n, so Validate cannot reject this cheaply. With a 64 Kibit
	// modulus, unpatched v3.0.5 spends tens of seconds in Validate. The bounds
	// check must refuse it first.
	p := oddWithBits(32768)
	n := new(big.Int).Mul(p, p)
	d := new(big.Int).Sub(n, big.NewInt(2))
	var key JSONWebKey
	err := json.Unmarshal(privateJWK(n, d, p, p), &key)
	if err == nil || !strings.Contains(err.Error(), "modulus too large") {
		t.Fatalf("oversized private modulus: got %v, want modulus too large", err)
	}
}

func TestRSAPrivateJWKOversizedPrimesRejected(t *testing.T) {
	// The modulus is within bounds, but p and q are huge and do not multiply
	// to n. The cheap bit-length check must refuse them before Validate.
	n := oddWithBits(2048)
	huge := oddWithBits(65536)
	var key JSONWebKey
	err := json.Unmarshal(privateJWK(n, n, huge, huge), &key)
	if err == nil || !strings.Contains(err.Error(), "p * q != n") {
		t.Fatalf("oversized primes: got %v, want p * q != n", err)
	}
}

func TestRSAPrivateJWKMismatchedPrimesRejected(t *testing.T) {
	a, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	b, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var key JSONWebKey
	err = json.Unmarshal(privateJWK(a.N, a.D, a.Primes[0], b.Primes[1]), &key)
	if err == nil || !strings.Contains(err.Error(), "p * q != n") {
		t.Fatalf("mismatched primes: got %v, want p * q != n", err)
	}
}

func TestRSAJWKRoundTripAndSign(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(JSONWebKey{Key: priv, KeyID: "k1", Algorithm: string(RS256)})
	if err != nil {
		t.Fatal(err)
	}
	var parsed JSONWebKey
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("valid private JWK rejected: %v", err)
	}
	got, ok := parsed.Key.(*rsa.PrivateKey)
	if !ok || !got.Equal(priv) {
		t.Fatalf("private key changed in round trip")
	}

	pubData, err := json.Marshal(parsed.Public())
	if err != nil {
		t.Fatal(err)
	}
	var pub JSONWebKey
	if err := json.Unmarshal(pubData, &pub); err != nil {
		t.Fatalf("valid public JWK rejected: %v", err)
	}

	signer, err := NewSigner(SigningKey{Algorithm: RS256, Key: &parsed}, nil)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := signer.Sign([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	compact, err := obj.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	jws, err := ParseSigned(compact)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := jws.Verify(&pub)
	if err != nil || string(payload) != "hello" {
		t.Fatalf("verify: payload %q, err %v", payload, err)
	}
}
