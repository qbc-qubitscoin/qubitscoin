# QubitsCoin (QBC) — Engineering & Development Rules

> **MANDATORY PROJECT POLICY**: All contributors, core developers, and AI assistants working on QubitsCoin must strictly adhere to the following rules. No pull request or commit will be accepted unless all rules are satisfied.

---

## 1. Test-Driven Development (TDD) — First Principle

### Rule Statement
> **Before writing any logic or implementation code, you MUST first write the unit tests.**

### The TDD Workflow:
1. **Red Phase (Write Tests First)**:
   - Define the public interface and expected behavior.
   - Write comprehensive unit tests (table-driven tests covering edge cases, happy paths, zero values, and error conditions).
   - Execute the test suite to verify that the tests fail or fail to compile as expected.
2. **Green Phase (Implement Minimal Logic)**:
   - Implement only the minimal logic necessary to make the tests pass.
   - Do not write speculative or dead code that is not verified by tests.
3. **Refactor Phase (Clean and Optimize)**:
   - Refactor for clarity, performance, and memory efficiency.
   - Run the test suite to ensure no regressions occur.
   - Ensure 100% test coverage is maintained.

---

## 2. Developer Documentation & "Why & How" Notes

### Rule Statement
> **Before or alongside any architectural or feature change, you MUST write detailed notes explaining WHY and HOW the logic works in the `developer_notes/` directory.**

### Requirements for Developer Notes:
- Every subsystem, consensus mechanism, smart contract, and crypto module must have a corresponding document in `developer_notes/`.
- Notes must document:
  1. **Why**: The problem being solved, cryptographic guarantees, and security considerations.
  2. **How**: Step-by-step state transition walkthrough, memory layouts, formulas, and wire formats.
  3. **Invariants**: Explicit mathematical invariants that cannot be violated (e.g., $x \cdot y = k$, fee conservation, halving curves).
  4. **Error Handling**: Detailed taxonomy of all errors returned and recovery strategies.

---

## 3. 100% Test Coverage Standard

### Rule Statement
> **All packages in `internal/` must achieve and maintain 100% statement test coverage.**

- Deterministic table-driven unit tests for internal state logic.
- BDD specifications using Ginkgo v2 and Gomega in `test/bdd/` for end-to-end integration flows.
- Concurrency and race-safety tests for shared state (`StateDB`, `Mempool`).
- Fault-injection and error path coverage for all I/O and networking boundaries.
- No dead code: If a branch is mathematically impossible to reach, remove the dead code rather than leaving untested paths.

---

## 4. Repository Cleanliness & Artifact Hygiene

### Rule Statement
> **Never commit binaries, build outputs, coverage dumps, or temporary test scratch files.**

- **Forbidden in commits**:
  - `*.out`, `*.prof`, `coverage*`, `cover*`, `*.html`
  - `*.exe`, `*.dll`, `*.so`, `*.dylib`
  - Temporary scratch files (`test_*.go`, `*.tmp`, `*.bak`, `*.wasm` in the root)
  - Unrelated or third-party explanatory documents (e.g., `bitcoin-explained.md`)
- Ensure `.gitignore` is strictly enforced.
- Run `git status` before committing to verify zero untracked garbage files.

---

## 5. Pure Go & Zero CGO Dependency

### Rule Statement
> **QubitsCoin core must compile and run on Windows, Linux, and macOS without requiring a C compiler (`CGO_ENABLED=0`).**

- Use Cloudflare Circl for post-quantum primitives (ML-DSA-65, ML-KEM-768).
- Use Wazero for the WebAssembly runtime (100% pure Go WebAssembly interpreter/compiler).
- Use Syndtr GoLevelDB for pure Go persistent storage.
