# Implementation Plan: Nostr Relay with Group Projections

**Branch**: `001-nostr-relay-groups` | **Date**: 2026-02-05 | **Spec**: /Users/peterwei/wokspace/s-city/specs/001-nostr-relay-groups/spec.md
**Input**: Feature specification from `/specs/001-nostr-relay-groups/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Build a Nostr relay that accepts and serves standard events, projects NIP-29
relay-based group state into queryable tables, and enforces deletion handling,
rate limits, and vetted-join approval rules while preserving user sovereignty,
federation, and privacy constraints.

## Technical Context

**Language/Version**: Go 1.25.5  
**Primary Dependencies**: khatru (latest upstream), go-nostr, pgx (PostgreSQL driver)  
**Storage**: PostgreSQL 18.1  
**Testing**: go test (unit + integration)  
**Target Platform**: Linux server  
**Project Type**: single  
**Performance Goals**: p95 publish/query under 2s; group projection within 5s  
**Constraints**: geohash level 6 max; no movement history; PoW/rate limiting  
**Scale/Scope**: initial target 10k concurrent connections; 5k events/min sustained

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] Identity and key handling are user-owned; no key escrow or recovery.
- [x] All message/call content is end-to-end encrypted; infrastructure cannot decrypt.
- [x] Federation and operator choice are preserved; no centralized control assumptions.
- [x] System behavior supports conversation, not enforcement of real-world actions.
- [x] Cost-imposing actions include proportional cost controls (PoW or equivalent).
- [x] Location handling respects geohash level 6 max precision and no movement history.
- [x] Sidecar fail-closed behavior and policy enforcement are defined where applicable (not in scope for this relay feature).

## Project Structure

### Documentation (this feature)

```text
specs/001-nostr-relay-groups/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
src/
├── models/
├── services/
├── relay/
├── storage/
└── lib/

tests/
├── contract/
├── integration/
└── unit/
```

**Structure Decision**: Single Go service with shared libraries and explicit
service layers; integration tests cover database and relay protocol behavior.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |

## Constitution Compliance Review Notes (2026-02-05)

- CC-001 (user-owned keys): server stores only public keys and signatures;
  private key handling and escrow are not implemented.
- CC-002 (E2EE): relay persists opaque event content; no decryption path exists
  in services, storage, or routes.
- CC-003 (federation/operator choice): service exposes standard Nostr event
  semantics with independent relay operation and no centralized dependency.
- CC-004 (location precision/privacy): group geohash values are truncated and
  validated to max precision level 6 in projection handling and repository
  writes.
- CC-005 (proportional cost controls): per-pubkey burst+sustained rate limiting
  and PoW thresholds by event kind are enforced during ingest.
