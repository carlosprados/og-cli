# Handoff: MQTT TLS — broker omits the intermediate certificate

> Status: **deferred to v1.1+**. v1.0 ships with `--insecure` defaulting to **true**
> for the MQTT south client. This document is the pickup brief for a future session
> to do it properly. Have `pkg/opengate/mqtt.go` open alongside this.

## TL;DR

The OpenGate MQTT broker (`api.opengate.es:8883`) presents a **valid Let's Encrypt
certificate** — NOT a private CA, as we first assumed. The TLS failure
(`x509: certificate signed by unknown authority`) happens because **the broker does
not send the intermediate certificate** in the TLS handshake (it sends only the leaf).
Go cannot build the chain leaf → R13 (intermediate) → ISRG Root X1 (trusted root)
without the intermediate, so verification fails. Today we work around this with
`InsecureSkipVerify`, which is a v1.0 wart we want gone.

## Evidence (gathered 2026-06-16)

```
# MQTT 8883 — sends ONLY the leaf (no intermediate):
$ echo | openssl s_client -connect api.opengate.es:8883 -servername api.opengate.es -showcerts
 0 s:CN = api.opengate.es
   i:C = US, O = Let's Encrypt, CN = R13     <-- issuer is the LE R13 intermediate
 (no chain entry 1 — the R13 intermediate is NOT sent)

# leaf is a normal public LE cert:
issuer = C=US, O=Let's Encrypt, CN=R13
subject = CN=api.opengate.es
notBefore=Apr  7 2026  notAfter=Jul  6 2026

# HTTPS 443 — works in Go because it DOES send the full chain (leaf + intermediate).
issuer = C=US, O=Let's Encrypt, CN=R13   (same leaf, but 443 serves the chain)
```

So the north HTTPS plane validates fine (full chain served); only the MQTT broker
is misconfigured to omit the intermediate.

## The real fix (preferred): fix the broker

Configure the MQTT broker (mosquitto/EMQX/whatever fronts 8883) to present the
**full certificate chain** (leaf + R13 intermediate), exactly as the 443 endpoint
already does. This is the correct, zero-maintenance fix — usually pointing the broker
at `fullchain.pem` instead of `cert.pem`. Once done, og can default to TLS-verify-ON
with the system root store and **no client-side changes are needed** beyond flipping
the defaults (below). This is an Amplía-infra task.

## The client-side workaround (if the broker can't be fixed)

Embed the Let's Encrypt intermediate(s) and add them to the cert pool so Go can build
the chain itself. In `pkg/opengate/mqtt.go` `NewMQTTClient`:

```go
// instead of opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: insecure})
pool, _ := x509.SystemCertPool()
if pool == nil { pool = x509.NewCertPool() }
pool.AppendCertsFromPEM(letsEncryptIntermediatesPEM) // go:embed-ed PEM
opts.SetTLSConfig(&tls.Config{RootCAs: pool, ServerName: host})
```

Caveats: LE rotates intermediates (R10–R14, ~3-year validity), so the embedded PEM
needs occasional refresh — embed several current intermediates to reduce churn, and
add a test that the bundle isn't expired. This is why the broker fix is preferred.

## What v1.0 should do regardless

- Keep `--insecure` as an escape hatch (already present).
- Add a **`--ca-file <path>`** flag so an operator can supply the chain/CA without a
  rebuild — cheap, and removes the need to ship `--insecure=true` for sites that have
  the chain. (Not yet implemented; small.)
- Once the broker is fixed OR `--ca-file`/embed lands, **flip the defaults**:
  `--insecure` → false. The flag plumbing lives in `cmd/iot.go` (shared MQTT flags
  loop) and the MCP defaults in `internal/mcp/tools_iot_mqtt.go` (`mqttConnect`,
  currently `insecure` defaults true).

## Code touchpoints

- `pkg/opengate/mqtt.go` — `NewMQTTClient(host, port, useTLS, insecure, ...)`, the
  `tls.Config{InsecureSkipVerify: insecure}` line.
- `cmd/iot.go` — the shared MQTT flags loop: `--tls` (default true), `--insecure`
  (default true), `--port`. Add `--ca-file` here.
- `internal/mcp/tools_iot_mqtt.go` — `mqttConnect` reads `tls`/`insecure` args
  (default true). Mirror any new `--ca-file` behaviour.

## Acceptance for v1.1

`og iot publish <dev> ... ` and the MCP MQTT tools connect to `api.opengate.es:8883`
**without `--insecure`**, verifying the chain (via fixed broker, `--ca-file`, or the
embedded intermediate), and reject a genuinely bad/self-signed cert.
