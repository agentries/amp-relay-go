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

## Review Checklist

### 1. 🎯 Correctness & Logic
- [ ] Business logic behaves as intended
- [ ] Edge cases handled (nil, empty, boundary values)
- [ ] Protocol compliance (AMP v5.0 spec)
- [ ] CBOR encoding/decoding correctness
- [ ] DID signature verification implemented correctly

### 2. 🔒 Security
- [ ] Input validation (untrusted data from network)
- [ ] No hardcoded credentials or secrets
- [ ] Proper cryptographic practices (Ed25519)
- [ ] Replay attack protection (timestamps, nonces)
- [ ] No unsafe pointer operations

### 3. ⚠️ Error Handling
- [ ] All errors checked (no ignored returns)
- [ ] Errors wrapped with context (`fmt.Errorf("...: %w", err)`)
- [ ] Appropriate error types (custom errors where needed)
- [ ] Graceful degradation on failures

### 4. 🔄 Concurrency Safety
- [ ] Proper mutex usage for shared state
- [ ] No data races (run with `-race` flag)
- [ ] Channel usage is correct (closed properly, no panics)
- [ ] Goroutine lifecycle managed (context cancellation)
- [ ] No goroutine leaks

### 5. 📊 Performance
- [ ] Efficient memory allocation (reuse buffers where possible)
- [ ] No unnecessary allocations in hot paths
- [ ] Appropriate use of sync.Pool for frequent allocations
- [ ] Proper connection pooling for WebSocket
- [ ] Batch operations where applicable

### 6. 📖 Readability & Idioms
- [ ] Idiomatic Go style (gofmt compliant)
- [ ] Clear naming (verbs for functions, nouns for types)
- [ ] Appropriate comments (why, not what)
- [ ] Package structure follows Go conventions
- [ ] Public API is minimal and well-documented

### 7. 🧪 Testability
- [ ] Unit tests exist for core logic
- [ ] Tests are deterministic (no flaky tests)
- [ ] Test coverage for error paths
- [ ] Interfaces used for testability (dependency injection)
- [ ] Table-driven tests where appropriate

### 8. 🏗️ Architecture (AMP-Specific)
- [ ] MessageStore interface properly abstracted
- [ ] Transport layer decoupled from application logic
- [ ] Protocol handlers are composable
- [ ] Configuration externalized (no magic constants)
- [ ] Logging is structured (with context)

---

## Output Format

1. **Summary**: Overall code quality assessment (1-2 sentences)

2. **Critical Issues** 🔴 (must fix before merge):
   - Issue description
   - Location (file:line or function name)
   - Suggested fix with code snippet

3. **Improvements** 🟡 (recommended):
   - Issue description
   - Location
   - Suggested improvement

4. **Suggestions** 🟢 (optional, nice-to-have):
   - Description
   - Rationale

5. **Positive Highlights** ✅:
   - What's done well

---

## Review Instructions

When invoking this command:

1. **Read** all Go files in the target (or entire project if no target)
2. **Analyze** each file against the checklist above
3. **Prioritize** security and correctness issues
4. **Provide** specific, actionable feedback with code examples
5. **Be constructive** - explain why changes improve the code

After review, if issues are found:
- Propose refactored code
- Wait for confirmation before applying changes
- Re-review after refactoring

Repeat until no critical issues remain.
