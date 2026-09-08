# QubitsCoin Bug Bounty Report - Testnet Alpha

**Phase**: 7
**Scope**: Core Client, Consensus Engine, QubitVM

## Overview
During the Testnet Alpha phase, a simulated bug bounty program was run targeting the core consensus logic, cryptographic primitives, and the `wazero`-based QubitVM.

## Findings Summary

| Severity | Count | Status |
|----------|-------|--------|
| Critical | 0     | N/A |
| High     | 1     | Resolved |
| Medium   | 2     | Resolved |
| Low      | 4     | Acknowledged |

## Resolved Issues

### High: RPC Keystore Password Leak via Docker Compose
**Description**: The `docker-compose.yml` was passing the keystore password via command-line arguments (`--password`), which caused the password to leak into process lists and `docker inspect` outputs.
**Remediation**: Removed CLI flag usage. The node now strictly reads the `QBC_PASSWORD` environment variable. (Fixed in `v0.5.0` pre-release).

### Medium: Unhandled Panics in Crypto Fuzzing
**Description**: Improper hex formatting in addresses or hashes could cause parsing panics if not gracefully caught.
**Remediation**: Added comprehensive fuzz testing to `internal/crypto` and `internal/core`. All parsing functions now return typed errors instead of panicking on malformed input.

### Medium: Node Healthcheck Used GET Instead of POST
**Description**: The docker healthcheck attempted to query the JSON-RPC API via an unauthenticated `GET` request, resulting in constant 405 Method Not Allowed errors.
**Remediation**: Re-wrote the healthcheck to use a properly formatted JSON-RPC `POST` payload.

## Conclusion
No critical consensus failure or cryptographic bypass vulnerabilities were discovered. The ML-DSA-65 signature integration and EIP-1559 fee model operated exactly as specified. The testnet is deemed stable for progression to Phase 8 (Wallet Development).
