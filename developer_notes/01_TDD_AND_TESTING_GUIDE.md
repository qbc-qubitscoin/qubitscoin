# Developer Note 01: Test-Driven Development (TDD) & Testing Methodology

## 1. Why TDD is Mandatory in QubitsCoin

In financial and blockchain systems, bugs cannot be "patched in production" without catastrophic risk, hard forks, or financial loss. QubitsCoin enforces **Test-Driven Development (TDD)** as a core engineering invariant:

1. **Bug Prevention Over Detection**: Writing tests first forces the developer to define clear, unambiguous boundaries, error states, and mathematical limits before writing code.
2. **Deterministic State Guarantees**: A blockchain state machine must be 100% deterministic. If two nodes produce differing state roots for the same transaction, the chain forks. TDD ensures every branch is tested for determinism.
3. **Refactoring Without Fear**: With 100% test coverage, core algorithms (such as base fee calculation or VM gas metering) can be aggressively optimized knowing the test suite will catch regressions immediately.

---

## 2. How TDD is Executed Step-by-Step

### Phase 1: Write the Failing Test (Red)
Before touching any `.go` file in `internal/`, create or open `[file]_test.go`:
- Write table-driven test cases with `struct` entries representing:
  - Valid input and expected state change.
  - Boundary conditions: Zero values (`0`), maximum integers (`math.MaxUint64`), empty slices, nil pointers.
  - Failure conditions: Insufficient balance, invalid nonce, short or tampered cryptographic signatures, malformed encoding.
- Run `go test ./...` to observe the test fail.

### Phase 2: Implement Minimal Code (Green)
- Implement only the minimal logic required to pass the test cases.
- Avoid speculative defensive logic that cannot be reached (e.g. checking conditions that are mathematically impossible).

### Phase 3: Refactor & Verify 100% Coverage (Refactor)
- Clean up variable names, optimize memory allocations.
- Execute coverage profiling:
  ```powershell
  go test -v "-coverprofile=coverage.out" ./internal/...
  go tool cover "-func=coverage.out"
  ```
- Verify statement coverage is at **100.0%**.

---

## 3. Testing Tiers in QubitsCoin

### 1. Unit Tests (Table-Driven)
Located directly alongside packages (`internal/core/*_test.go`, `internal/state/*_test.go`).
- Focus: Individual functions, state transitions, mathematical calculations.
- Pattern:
  ```go
  tests := []struct{
      name    string
      input   uint64
      want    uint64
      wantErr bool
  }{
      {"normal case", 100, 105, false},
      {"zero input", 0, 0, true},
  }
  ```

### 2. Behavior-Driven Development (BDD Specs)
Located in `test/bdd/` using **Ginkgo v2** and **Gomega**:
- Focus: End-to-end user workflows written in human-readable Given/When/Then style.
- Suites:
  - `transaction_bdd_test.go`: End-to-end wallet creation, transaction signing, mempool ingestion, block production, and account balance verification.
  - `lending_bdd_test.go`: Collateral deposit, borrowing against collateral, health factor calculation, and liquidation thresholds.
  - `dex_bdd_test.go`: Liquidity addition, constant product AMM swaps ($x \cdot y = k$), and 0.3% fee collection.

### 3. Concurrency & Race Condition Tests
Located in `internal/state/statedb_concurrent_test.go` and `internal/mempool/mempool_concurrent_test.go`:
- Uses `sync.WaitGroup` and hundreds of concurrent goroutines reading and writing to ensure mutexes and read/write locks prevent deadlocks and race conditions.

### 4. Fuzz Testing
Located in `internal/core/` and `internal/crypto/`:
- Native Go fuzzers (`FuzzNextBaseFee`, `FuzzTransactionValidate`, `FuzzHexToHash`) that inject randomized binary data to verify no panics occur on malformed network payloads.
