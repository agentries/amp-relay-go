# Language Decision: Go for AMP Relay Implementation

## Decision Summary

Based on Ryan Cooper's communications and technical analysis, **Go (Golang)** has been selected as the primary implementation language for the AMP Relay reference implementation.

## Decision Rationale

### From Ryan's Communications (UID 207)

Ryan confirmed the implementation language preference:

> **"Go Relay Server Prototype (RFC-003)"**
> - Language: **Go** (aligned with Ryan's preference)
> - Timeline: 4-week sprint (Feb 5 → Mar 5, 2026)
> - Deliverable: Working relay for Ryan review

### Technical Analysis

| Criteria | Go | Rust | TypeScript |
|----------|-----|------|------------|
| **CBOR Libraries** | ⭐⭐⭐ Excellent | ⭐⭐⭐ Good | ⭐⭐⭐ Good |
| **Ed25519** | ⭐⭐⭐ Native (crypto/ed25519) | ⭐⭐⭐ Excellent | ⭐⭐⭐ Available (tweetnacl) |
| **NaCl/Box** | ⭐⭐⭐ Native (x/crypto/nacl) | ⭐⭐⭐ Excellent | ⭐⭐⭐ Available |
| **Concurrency** | ⭐⭐⭐ Goroutines | ⭐⭐⭐ Async/await | ⭐⭐ Async/await |
| **Deployment** | ⭐⭐⭐ Single binary | ⭐⭐⭐ Single binary | ⭐⭐ Node runtime |
| **Dev Speed** | ⭐⭐⭐ Fast | ⭐⭐ Slower | ⭐⭐⭐ Fastest |
| **Memory Safety** | ⭐⭐ GC | ⭐⭐⭐ Compile-time | ⭐⭐ GC |

### Why Go Won

1. **Ryan's Preference**: Direct confirmation from the AMP specification author
2. **Concurrency Model**: Goroutines and channels ideal for relay's connection handling
3. **Library Ecosystem**: Mature CBOR (fxamacker/cbor/v2), native Ed25519
4. **Deployment**: Single static binary, easy cross-compilation
5. **Proven at Scale**: Many high-throughput network services use Go

### When to Use Other Languages

- **Rust**: High-performance relay when throughput demands justify learning curve
- **TypeScript**: Client libraries for web/browser-based agents
- **Python**: Rapid prototyping (not recommended for production relay)

## Current Implementation Status

### Completed
- [x] Project scaffolding
- [x] CBOR message encoding/decoding
- [x] WebSocket transport layer
- [x] DID-based authentication framework
- [x] Message signing/verification (Ed25519 + JWS)
- [x] NaCl box encryption
- [x] In-memory message store

### In Progress
- [ ] WebSocket auth handshake (RFC-002)
- [ ] DID document resolution over HTTP
- [ ] Message routing/forwarding
- [ ] Store-and-forward persistence

## References

- Ryan's Decision Email: `/research/lab/ryan-communications/2026-02-05_0347_impl_commitment_answers.md`
- Technical Analysis: `/research/lab/LAB_ARCHIVE/IMPLEMENTATION_LANGUAGE_ANALYSIS.md`
- Current Code: `/research/lab/amp-relay-go/`

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-02-05 | Go selected | Ryan's preference + practical factors |
| 2026-02-06 | Go confirmed | All core tests passing |

---
*Document Version: 1.0*
*Maintained by: The Lab Engineering Team*
