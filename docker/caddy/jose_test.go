package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"go.step.sm/crypto/jose"
)

// Caddy's PKI app reaches go-jose v3 through go.step.sm/crypto/jose. These
// tests exercise that path so they fail if the GHSA-mgcc-cww5-fp2q backport in
// third_party/go-jose-v3 stops being used.

// oversizedRSAJWK is a private RSA JWK with a 64 Kibit modulus where n = p*q
// and d < n, so rsa.PrivateKey.Validate cannot reject it cheaply. Without the
// bounds check, parsing it spends CPU that grows cubically with the key size.
func oversizedRSAJWK() string {
	p := new(big.Int).Lsh(big.NewInt(1), 32767)
	p.Or(p, big.NewInt(1))
	n := new(big.Int).Mul(p, p)
	d := new(big.Int).Sub(n, big.NewInt(2))
	enc := func(v *big.Int) string { return base64.RawURLEncoding.EncodeToString(v.Bytes()) }
	return fmt.Sprintf(`{"kty":"RSA","n":%q,"e":"AQAB","d":%q,"p":%q,"q":%q}`,
		enc(n), enc(d), enc(p), enc(p))
}

func wantModulusTooLarge(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("oversized RSA JWK accepted")
	}
	if msg := err.Error(); !strings.Contains(msg, "modulus too large") {
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		t.Fatalf("got %q, want modulus too large", msg)
	}
}

func TestStepJoseRejectsOversizedRSAJWK(t *testing.T) {
	var key jose.JSONWebKey
	wantModulusTooLarge(t, json.Unmarshal([]byte(oversizedRSAJWK()), &key))
}

func TestStepJoseRejectsOversizedEmbeddedJWK(t *testing.T) {
	// ACME clients embed their account key in the JWS protected header, so a
	// request body is enough to reach JWK parsing.
	enc := base64.RawURLEncoding.EncodeToString
	header := fmt.Sprintf(`{"alg":"RS256","jwk":%s}`, oversizedRSAJWK())
	token := enc([]byte(header)) + "." + enc([]byte("{}")) + "." + enc([]byte("sig"))

	_, err := jose.ParseJWS(token)
	wantModulusTooLarge(t, err)
}

func TestStepJoseRSAJWTRoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(jose.JSONWebKey{Key: priv, KeyID: "k1", Algorithm: jose.RS256})
	if err != nil {
		t.Fatal(err)
	}
	key, err := jose.ParseKey(data)
	if err != nil {
		t.Fatalf("valid RSA JWK rejected: %v", err)
	}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	token, err := jose.Signed(signer).Claims(jose.Claims{
		Subject:  "facet",
		Audience: jose.Audience{"caddy"},
		IssuedAt: jose.NewNumericDate(now),
		Expiry:   jose.NewNumericDate(now.Add(time.Minute)),
	}).CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := jose.ParseSigned(token)
	if err != nil {
		t.Fatal(err)
	}
	pub := key.Public()
	var claims jose.Claims
	if err := parsed.Claims(&pub, &claims); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := claims.Validate(jose.Expected{Subject: "facet", Audience: jose.Audience{"caddy"}, Time: now}); err != nil {
		t.Fatalf("claims: %v", err)
	}
}
