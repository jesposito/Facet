# Patched go-jose v3.0.5

This directory is a copy of `github.com/go-jose/go-jose/v3` v3.0.5 with the RSA JWK
security fix backported and an explicit Unicode parsing bound. `docker/caddy/go.mod` points the module at this copy
with a `replace` directive.

## Why it exists

[GHSA-mgcc-cww5-fp2q](https://github.com/go-jose/go-jose/security/advisories/GHSA-mgcc-cww5-fp2q)
affects go-jose v3 up to and including v3.0.5. A crafted RSA JWK can make
`rsa.PrivateKey.Validate` run for a long time, because go-jose passes the
modulus and primes to it without checking their size first. Upstream fixed this
in v4.1.6 and has not published a fixed v3 release.

Caddy reaches v3 through this import chain:

```
facet/caddy
  github.com/caddyserver/caddy/v2/modules/standard
  github.com/caddyserver/caddy/v2/modules/caddypki
  github.com/smallstep/certificates/authority
  go.step.sm/crypto/jose
  github.com/go-jose/go-jose/v3
```

On 2026-11-03 there was no compatible upstream path that removes v3:

- go.step.sm/crypto v0.91.1, the latest tag, still imports go-jose v3.
- go.step.sm/crypto master has moved `jose` to go-jose v4 (smallstep/crypto#1157),
  but that changes the API. smallstep/certificates v0.30.2 no longer compiles
  against it (for example, `jose.Expected` has no `Audience` field and the JWT
  builder has no `CompactSerialize`).
- smallstep/certificates master still requires go-jose v3 directly and pins
  go.step.sm/crypto v0.91.0.

## Source

- Upstream: https://github.com/go-jose/go-jose, tag `v3.0.5`, commit
  `be2f6548701d4ba561cd61c7d4a2f1289b66c273`
- Module checksum: `h1:BLLJWbC4nMZOfuPVxoZIxeYsn6Nl2r1fITaJ78UQlVQ=`
- go.mod checksum: `h1:5b+7YgP7ZICgJDBdfjZaIt+H/9L9T/YQrVfLAMboGkQ=`

The files came from the checksum-verified module via `go mod vendor`. Every
copied file, and `go.mod`, matches the git blob hash of the same path at the
upstream tag. `go mod vendor` only copies packages that the build uses, so the
upstream `_test.go` files, `json/testdata`, `jose-util/`, `go.sum` and
`.github/` are not included.

`LICENSE` (Apache License 2.0) and `json/LICENSE` (BSD-style, from the Go
authors) are unchanged.

## Changes from v3.0.5

`jwk.go` adds the RSA bounds checks from upstream commit
[a0111f8](https://github.com/go-jose/go-jose/commit/a0111f82742fa92bceec9ede40845e73319e6230)
(go-jose v4.1.6). The code is the same as v4.1.6's `rsaPublicKey` and
`rsaPrivateKey`:

- `MaxRSAModulusBits`, default 4096, caps the modulus of RSA public and
  private JWKs.
- Private RSA JWKs must have `p * q == n`. A cheap bit-length test runs first,
  so oversized primes are rejected before the multiplication.

Both checks run before `rsa.PrivateKey.Validate`. The rest of a0111f8 (header
algorithm ordering, rejecting private key members in embedded `jwk`/`epk`
headers) is not part of this advisory and has not been backported.

`json/decode.go` parses the four hexadecimal digits in a JSON Unicode escape
with a 16-bit bound instead of 64 bits. Four digits already limit the value to
65535; making that bound explicit preserves behavior and makes the conversion
to a rune provably safe for static analysis.

`jwk_rsa_bounds_test.go` is new. It checks the size limit, the `p * q == n`
check, and an ordinary RSA JWK round trip with RS256 signing.
`docker/caddy/jose_test.go` runs the same rejection through
`go.step.sm/crypto/jose`, including a JWK embedded in a JWS header. Against
unpatched v3.0.5, the 64 Kibit key in that test kept one parse busy for about
20 seconds before it failed.

```diff
@@ -481,13 +481,25 @@ func (k *JSONWebKey) Valid() bool {
 	return true
 }
 
+// MaxRSAModulusBits is the largest RSA modulus size, in bits, that will be
+// accepted when parsing a JWK. Applications that need larger keys can raise
+// this limit.
+//
+// If modifying MaxRSAModulusBits, do it before any parsing occurs.
+var MaxRSAModulusBits = 4096
+
 func (key rawJSONWebKey) rsaPublicKey() (*rsa.PublicKey, error) {
 	if key.N == nil || key.E == nil {
 		return nil, fmt.Errorf("go-jose/go-jose: invalid RSA key, missing n/e values")
 	}
 
+	n := key.N.bigInt()
+	if n.BitLen() > MaxRSAModulusBits {
+		return nil, fmt.Errorf("go-jose/go-jose: invalid RSA key, modulus too large")
+	}
+
 	return &rsa.PublicKey{
-		N: key.N.bigInt(),
+		N: n,
 		E: key.E.toInt(),
 	}, nil
 }
@@ -628,16 +640,23 @@ func (key rawJSONWebKey) rsaPrivateKey() (*rsa.PrivateKey, error) {
 		return nil, fmt.Errorf("go-jose/go-jose: invalid RSA private key, missing %s value(s)", strings.Join(missing, ", "))
 	}
 
+	n := key.N.bigInt()
+	if n.BitLen() > MaxRSAModulusBits {
+		return nil, fmt.Errorf("go-jose/go-jose: invalid RSA key, modulus too large")
+	}
+
+	p, q := key.P.bigInt(), key.Q.bigInt()
+	if p.BitLen()+q.BitLen() > n.BitLen()+1 || new(big.Int).Mul(p, q).Cmp(n) != 0 {
+		return nil, errors.New("go-jose/go-jose: invalid RSA private key, p * q != n")
+	}
+
 	rv := &rsa.PrivateKey{
 		PublicKey: rsa.PublicKey{
-			N: key.N.bigInt(),
+			N: n,
 			E: key.E.toInt(),
 		},
-		D: key.D.bigInt(),
-		Primes: []*big.Int{
-			key.P.bigInt(),
-			key.Q.bigInt(),
-		},
+		D:      key.D.bigInt(),
+		Primes: []*big.Int{p, q},
 	}
 
 	if key.Dp != nil {
```

## Running the tests

Run the tests from the parent `docker/caddy` module, not from inside this
`third_party` directory:

```
cd docker/caddy
go test github.com/go-jose/go-jose/v3/...
```

This resolves the patched copy through the `replace` directive in
`docker/caddy/go.mod` and builds it against the same upgraded dependency
versions Caddy actually links, which is the point of the exercise. The run
covers `jwk_rsa_bounds_test.go` (the backported bounds checks) and, via
`docker/caddy/jose_test.go`, the same rejection through
`go.step.sm/crypto/jose`.

Do not try to run the tests as a standalone nested module from within
`third_party/go-jose-v3`. That copy has no `go.sum` of its own (`go mod vendor`
does not produce one for a vendored package), so an isolated `go test` there
fails resolving dependencies. The parent module owns the checksums, so invoking
the tests from `docker/caddy` is both the working and the representative path.

## Effect on Caddy

Facet's shipped `docker/Caddyfile` does not use `acme_server`, `tls internal`
or other PKI features, but the code is compiled into the binary, and
self-hosted configurations can turn those features on. With this patch, RSA
JWKs over 4096 bits are rejected wherever Caddy's internal CA parses them, for
example ACME account keys embedded in request headers. RSA keys of 4096 bits
or less, and EC and Ed25519 keys, parse as before.

## Removing this copy

Delete this directory, the `replace` line in `docker/caddy/go.mod`, and the
`third_party` COPY in `docker/Dockerfile` once the selected
smallstep/certificates and go.step.sm/crypto releases no longer import go-jose
v3. Confirm with `go mod why -m github.com/go-jose/go-jose/v3`. The `replace`
only matches v3.0.5. If a dependency upgrade selects a different v3 version,
the replace stops applying, and that version needs checking against this
advisory.
