# 08: Auto-Upgrade and On-Chain Upgrade Scheduler

## 1. Architectural Overview

The internal/upgrade package implements autonomous, quantum-resistant node binary updates and scheduled on-chain network hardfork coordination.

`
┌────────────────────────────────────────────────────────┐
│                   GitHub Releases API                  │
│       (Tagged Release + Assets + .sha3sum Digest)      │
└───────────────────────────┬────────────────────────────┘
                            │ (HTTPS GET)
                            ▼
┌────────────────────────────────────────────────────────┐
│                   Release Fetcher                      │
│      - Matches OS/Architecture (e.g. windows-amd64)    │
│      - Parses Semantic Version (vX.Y.Z)                │
│      - Fetches SHA-3-256 Checksum Digest               │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│                   Streaming Downloader                 │
│      - MultiWriter streams payload to temp file        │
│      - Computes SHA-3-256 hash in flight               │
│      - Sets executable permissions (0755)              │
└───────────────────────────┬────────────────────────────┘
                            │ Verified binary
                            ▼
┌────────────────────────────────────────────────────────┐
│                   Atomic Applier                       │
│      - Renames running binary -> binary.old            │
│      - Moves new binary into place                     │
│      - Re-executes node process with identical args    │
│      - Rolls back on failure                           │
└────────────────────────────────────────────────────────┘
`

---

## 2. Key Components

### 2.1 Release Downloader (downloader.go)
- **Streams without buffering whole binaries in RAM**: Uses io.MultiWriter(tmpFile, sha3Hasher).
- **SHA-3-256 Verification**: Guarantees binary integrity before applying.
- **Atomic Destination**: Places temp files on the same filesystem partition as the running executable to ensure atomic os.Rename operations.

### 2.2 Atomic Applier (pplier.go)
- **Safe Replacement**: Moves <exe> to <exe>.old, then moves the new temp file to <exe>.
- **Rollback Protection**: If placing the new binary fails, automatically restores <exe>.old back to <exe>.
- **Cross-Platform Re-exec**: Spawns the new process via eExec (cmd.Start() on Windows with process detached, syscall.Exec on POSIX systems) and exits cleanly.
- **Startup Cleanup**: On initialization, cleanOldBinary() removes any residual <exe>.old artifacts.

### 2.3 On-Chain Upgrade Scheduler (scheduler.go)
- Coordinates planned protocol upgrades via threshold validator multi-signatures.
- A proposal requires:
  1. TargetHeight: Activation block height.
  2. TargetVersion: Target semantic version string.
  3. BinaryHash: Expected SHA-3-256 digest of the new binary.
- Proposals reach quorum when validated by > 2/3 of the active validator set.
- Activates automatically in OnBlock(height).

---

## 3. Testing and 100% Coverage Patterns

To test OS-level executable updates, re-execution, and HTTP streaming without destabilizing test runners:
1. **Inverted Injected Hooks**:
   - osExecutable: Simulates arbitrary executable paths and lookup errors.
   - createTemp: Injects failures during temporary file creation.
   - closeFile: Exercises flush and synchronization failure branches.
   - chmod: Validates permission error handling.
   - eExecFunc & osExit: Intercepts subprocess spawning and termination to prevent premature test runner exits.
2. **Abrupt TCP Drop Tests**:
   - Uses http.Hijacker to abruptly terminate TCP connections mid-stream to verify robust handling of network drops during io.Copy and io.ReadAll.
3. **Mock GitHub Releases Server**:
   - Emulates asset matrices, missing assets, checksum mismatches, HTTP 500s, and corrupted JSON tag names.
