# Go Code Review Command

## Usage
```
/project:review-code [file_or_directory]
```

## Project Context

This is the **AMP Reference Relay** - a Go implementation of the Agent Messaging Protocol (AMP) v5.0 for AI agent-to-agent communication.

### Key Design Principles
- **DID-Native**: Agents identified by Decentralized Identifiers
- **CBOR Encoding**: Binary protocol (RFC 8949), not JSON
- **Signature-Based**: All messages cryptographically signed (Ed25519)
- **Three-Layer Architecture**: Transport → Security → Application
- **Async-First**: No assumption of synchronous request-response

### Code Standards (Ryan Standard)
- **MUST**: Use CBOR tags for all Message fields
- **MUST**: Implement MessageStore interface to keep storage layer abstract
- **MUST**: All handshakes must include DID signature verification
- **FORBIDDEN**: Direct storage logic outside internal/ directory

---

## Review Dimensions

### 1. Change Summary & Motivation (What / Why)
- What changed? Summarize the scope of the diff (new feature, bug fix, refactor, config change, etc.)
- Why was this change made? Link to issue, design doc, or explain the motivation
- Is the change self-contained, or part of a larger effort?
- Does the commit message / PR description accurately reflect the change?

### 2. Design / Architecture & System Integration
- [ ] Consistent with the Three-Layer Architecture (Transport → Security → Application)
- [ ] MessageStore interface properly abstracted; no storage logic outside internal/
- [ ] Module boundaries and package structure are respected
- [ ] Public API surface is minimal and intentional
- [ ] Dependencies between packages flow in the correct direction
- [ ] New interfaces/abstractions are justified (not speculative)
- [ ] If introducing a new component, does it fit the existing system design?

### 3. Functional Correctness & Edge Cases (incl. Concurrency / Race Conditions)
- [ ] Business logic behaves as intended for the happy path
- [ ] Edge cases handled: nil, empty, zero-value, boundary values, overflow
- [ ] Protocol compliance with AMP v5.0 spec
- [ ] CBOR encoding/decoding correctness (tag usage, field ordering)
- [ ] DID signature verification implemented correctly
- [ ] Proper mutex usage for shared state; no unprotected concurrent access
- [ ] No data races (verify with `go test -race`)
- [ ] Channel usage is correct (closed properly, no sends on closed channels)
- [ ] Goroutine lifecycle managed (context cancellation, no leaks)
- [ ] Error handling: all errors checked, wrapped with context (`fmt.Errorf("...: %w", err)`)
- [ ] Graceful degradation on failures; appropriate error types

### 4. Complexity & Maintainability
- [ ] No over-engineering: abstractions are earned, not speculative
- [ ] Single-use helpers or premature generalizations avoided
- [ ] Can another engineer understand this code without the author explaining it?
- [ ] Functions/methods are focused (single responsibility, reasonable length)
- [ ] Configuration externalized; no magic constants
- [ ] No unnecessary indirection or wrapper layers

### 5. Tests & Evidence
- [ ] Unit tests exist for new/changed core logic
- [ ] Tests cover error paths, not just happy paths
- [ ] Tests are deterministic (no flaky tests, no timing dependencies)
- [ ] Tests can effectively *fail* — they actually assert meaningful invariants
- [ ] Table-driven tests used where appropriate
- [ ] Interfaces used for testability (dependency injection)
- [ ] Integration tests for cross-component interactions where needed
- [ ] If a bug fix, is there a regression test that would have caught it?

### 6. Readability / Naming / Comments / Documentation
- [ ] Clear naming: verbs for functions, nouns for types, consistent terminology
- [ ] Comments explain *why*, not *what*
- [ ] Public API has godoc-style documentation
- [ ] No dead code, commented-out blocks, or TODO without issue references
- [ ] If behavior changes, are README / docs / examples updated?

### 7. Security & Performance
**Security:**
- [ ] Input validation for untrusted data from network
- [ ] No hardcoded credentials or secrets
- [ ] Proper cryptographic practices (Ed25519, constant-time comparisons)
- [ ] Replay attack protection (timestamps, nonces)
- [ ] No unsafe pointer operations without justification
- [ ] Injection risks considered (command, path traversal, etc.)

**Performance:**
- [ ] No unnecessary allocations in hot paths
- [ ] Efficient memory allocation (buffer reuse, sync.Pool where appropriate)
- [ ] Proper connection pooling for WebSocket
- [ ] No O(n^2) or worse algorithms where O(n log n) or better is feasible
- [ ] Batch operations where applicable

### 8. Go Style & Tooling
- [ ] `gofmt` / `goimports` compliant
- [ ] Passes `go vet` and `staticcheck` without warnings
- [ ] Follows Effective Go and Go Code Review Comments conventions
- [ ] Package structure follows Go conventions (internal/, cmd/, etc.)
- [ ] Structured logging with context (not fmt.Println)

### 9. Review Scope & Supplementary Reviewers
After completing the review, explicitly state:
- **Covered**: Which dimensions were fully reviewed
- **Not covered / Limited**: Which dimensions need deeper expertise
- **Recommended additional reviewers**:
  - Security reviewer (if crypto, auth, or input validation changes)
  - Concurrency reviewer (if goroutines, channels, or shared state changes)
  - Protocol reviewer (if AMP message format or handshake changes)
  - Performance reviewer (if hot path or high-throughput changes)

### 10. RFC / Spec Alignment
- [ ] Map changed code to relevant AMP v5.0 spec sections (cite chapter/section)
- [ ] If implementation deviates from spec, document the deviation and rationale
- [ ] If spec is ambiguous, note the interpretation chosen and flag for discussion
- [ ] Reference materials: `research/lab/LAB_ARCHIVE/` for spec details

---

## Output Format

Structure every review output as follows:

### 1. Summary
- What this change does and why (1-3 sentences)
- Overall assessment: Ready to merge / Needs revision / Needs major rework

### 2. Blockers (must fix before merge)
> Issues that are incorrect, insecure, or will cause production problems.

For each issue:
- **[B-n]** Description of the problem
- **Location**: `file:line` or function name
- **Why it matters**: Impact if not fixed
- **Suggested fix**: Code snippet or concrete recommendation

### 3. Majors (strongly recommended)
> Issues that significantly affect maintainability, correctness risk, or design.

For each issue:
- **[M-n]** Description
- **Location**: `file:line`
- **Suggestion**: How to improve

### 4. Minors (recommended)
> Style, readability, or minor improvements.

For each issue:
- **[m-n]** Description
- **Location**: `file:line`
- **Suggestion**: How to improve

### 5. Nits (optional, take-or-leave)
> Formatting, naming preferences, trivial cleanups.

- **[N-n]** Description — `file:line`

### 6. Questions
> Points that need clarification from the author before the reviewer can approve.

- **[Q-n]** Question — `file:line` (if applicable)

### 7. Positives
> Things done well that should be kept or that others can learn from.

- What's done well and why it's good

### 8. Review Scope
- **Covered**: [list of dimensions reviewed]
- **Supplementary reviewers needed**: [role — reason]

---

## Review Instructions

When invoking this command:

1. **Read** all Go files in the target (or entire project if no target specified)
2. **Understand motivation**: Read PR description, commit messages, or linked issues to understand the *why*
3. **Analyze** each file against all 10 review dimensions above
4. **Prioritize**: Security and correctness issues first, then design, then style
5. **Cite locations**: Always reference `file:line` for every finding
6. **Be specific and actionable**: Provide concrete code examples for fixes
7. **Explain the reasoning**: State *why* a change improves the code, not just *what* to change
8. **Check RFC alignment**: Map changes to AMP v5.0 spec sections where applicable

After review, if Blockers or Majors are found:
- Propose refactored code for each
- Wait for author confirmation before applying changes
- Re-review after refactoring

Repeat until no Blockers remain.
