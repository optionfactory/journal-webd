# Changelog

## 1.3 (2026-11-23)

### Fixed
- With `auth_type` `basic`, the websocket stream rejected valid basic credentials, so the UI
  could not stream logs. Broken since 1.1.

### Changed
- In `proxy_mode` `edge`, `X-Forwarded-For` is trusted only when the request comes from a
  loopback or private address (Fiber v3 requires an explicit trusted proxy list). The header
  is used only for the client address in the access log.
- Releases ship a `SHA256SUMS` file next to the binary.
- Upgraded to Fiber v3 (v2 is deprecated upstream) and golang-jwt v5. Accepted JWT signing
  methods are unchanged: RS256, RS384, RS512.
- Bumped indirect dependencies: fasthttp 1.80.0, x/net 0.63.0, x/sys 0.50.0.

## 1.2 (2026-11-15)

### Security
- Bumped dependencies to fix GO-2026-6888, GO-2026-6865 and GO-2025-4208.

### Changed
- Built with Go 1.27.4.

## 1.1 (2024-01-17)

### Changed
- Stream request fields renamed to camelCase: `rangeLines`, `rangePeriod`, `rangeMinutes`.
- A period range accepts `since` or `until` alone; at least one is required.
- Errors close the websocket with a close code and reason instead of being only logged:
  policy violation for authentication failures, internal server error otherwise.
- The startup log reports the listen address and protocol.

## 1.0 (2024-01-11)

First release.
