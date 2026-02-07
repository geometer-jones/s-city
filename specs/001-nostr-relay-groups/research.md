# Phase 0 Research: Nostr Relay with Group Projections

## Decision: Go runtime version

- **Decision**: Use Go 1.25.5 (latest 1.25.x patch release).
- **Rationale**: The Go release history lists 1.25.5 as the most recent patch
  for Go 1.25, with security fixes and bug fixes; using the latest patch
  reduces known vulnerabilities while keeping compatibility with 1.25.x.
- **Alternatives considered**:
  - Go 1.24.x: older supported release, but not the latest.

Sources:
- https://go.dev/doc/devel/release

## Decision: PostgreSQL version

- **Decision**: Use PostgreSQL 18.1 (current minor for major 18).
- **Rationale**: PostgreSQL versioning policy recommends the current minor
  release; the release notes identify 18.1 as current for major 18.
- **Alternatives considered**:
  - PostgreSQL 17.7: supported, but not the latest major.

Sources:
- https://www.postgresql.org/support/versioning/
- https://www.postgresql.org/docs/release/18.1/

## Decision: Relay framework (khatru)

- **Decision**: Use khatru with the latest tagged module version from
  `github.com/fiatjaf/khatru` (pin via `@latest`).
- **Rationale**: pkg.go.dev indicates khatru has tagged versions; using
  `@latest` ensures the newest tagged release is selected while remaining
  reproducible.
- **Alternatives considered**:
  - Building a relay from scratch: higher effort and risk.
  - Other relay frameworks: would diverge from the requested khatru use.

Sources:
- https://pkg.go.dev/github.com/fiatjaf/khatru

## Decision: Group rules and event semantics (NIP-29)

- **Decision**: Follow NIP-29 for group identifiers, `h` tags, and group
  management event kinds (join requests, moderation events, and metadata).
- **Rationale**: NIP-29 defines relay-based group semantics and event kinds that
  align with the requested state projection tables.
- **Alternatives considered**:
  - Custom group semantics: would reduce interoperability.

Sources:
- https://nips.nostr.com/29
