# Feature Specification: Nostr Relay with Group Projections

**Feature Branch**: `001-nostr-relay-groups`  
**Created**: 2026-02-05  
**Status**: Draft  
**Input**: User description: "Build a Nostr relay with standard events plus NIP-29\n+group state projections aligned to the provided schema."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Publish and Retrieve Standard Events (Priority: P1)

As a client, I can publish standard Nostr events to the relay and later query
for them by author, kind, time range, or tags so that conversations can be
coordinated and discovered.

**Why this priority**: It is the minimum viable behavior for any relay and the
foundation for all other capabilities.

**Independent Test**: Can be fully tested by publishing a valid event and
retrieving it via multiple filters while confirming invalid events are rejected.

**Acceptance Scenarios**:

1. **Given** a valid event with tags and signature, **When** it is submitted,
   **Then** it is stored and retrievable by author and kind.
2. **Given** invalid or duplicate events, **When** they are submitted, **Then**
   they are rejected and do not appear in queries.

---

### User Story 2 - Manage Group State Projections (Priority: P2)

As a group operator, I can create and update group metadata, roles, membership,
bans, invites, join requests, and group events so that group state is queryable
without replaying all history.

**Why this priority**: Group management is the core differentiator and is
required for NIP-29 group workflows.

**Independent Test**: Can be fully tested by publishing a sequence of group
related events and confirming the projected group state matches expected values.

**Acceptance Scenarios**:

1. **Given** a sequence of group events, **When** they are processed in order,
   **Then** the projected group tables reflect the latest state for that group.
2. **Given** a non-admin member, **When** they attempt to modify group metadata
   or roles, **Then** the update is rejected.
3. **Given** a group marked vetted, **When** a join request is submitted,
   **Then** approval is required before membership is granted.

---

### User Story 3 - Respect Deletions and Visibility (Priority: P3)

As a client, I can delete events and see that deletions are reflected in queries
and projections so that stale or revoked content is not served as active.

**Why this priority**: Deletions are part of standard relay behavior and prevent
inconsistent state.

**Independent Test**: Can be fully tested by submitting a delete request and
verifying the event no longer appears in active queries or group projections.

**Acceptance Scenarios**:

1. **Given** an existing event, **When** a valid deletion is submitted,
   **Then** the event is marked deleted and excluded from active results.

---

### Edge Cases

- Out-of-order group events arrive after newer updates for the same group.
- A user is banned and later re-invited before the ban expires.
- Invite codes exceed max usage or are used after expiration.
- Duplicate events or tags are submitted repeatedly.
- Events reference unknown groups or roles.
- A vetted group receives a join request without approval.
- Two group events with the same timestamp conflict.

## Requirements *(mandatory)*

### Constitution Constraints *(mandatory)*

- **CC-001**: Feature MUST preserve user-owned keys with no escrow/recovery.
- **CC-002**: Feature MUST keep content end-to-end encrypted; infrastructure
  cannot decrypt.
- **CC-003**: Feature MUST remain compatible with federation and operator
  choice.
- **CC-004**: Feature MUST NOT exceed geohash level 6 precision or store
  movement history.
- **CC-005**: Cost-imposing actions MUST include proportional cost controls.

### Functional Requirements

- **FR-001**: Relay MUST accept, validate, and store standard Nostr events and
  serve them through filtering by author, kind, time range, and tags.
- **FR-002**: Relay MUST reject invalid events, including bad signatures and
  malformed tags.
- **FR-003**: Relay MUST track deletions and exclude deleted events from active
  query results.
- **FR-012**: Deletions MUST remove items from active group projections while
  remaining queryable as deleted records.
- **FR-004**: Relay MUST project group state from group-related events into
  queryable group management records (metadata, roles, members, bans, invites,
  join requests, and group events).
- **FR-005**: Relay MUST keep group state consistent with the latest valid
  updates per group.
- **FR-009**: Relay MUST restrict group metadata, role, membership, ban, and
  invite changes to authorized group owner/admin roles.
- **FR-010**: When a group is marked vetted, the relay MUST require approval
  for join requests before membership is granted.
- **FR-011**: Group state conflicts MUST resolve by last-write-wins using the
  latest `created_at` timestamp.
- **FR-006**: Relay MUST enforce proportional cost controls for actions that
  impose shared infrastructure cost.
- **FR-013**: Rate limiting MUST allow short bursts followed by sustained
  limits per pubkey.
- **FR-007**: Relay MUST preserve privacy by avoiding storage of movement
  history and limiting location precision when group metadata includes location.
- **FR-008**: Relay MUST operate on currently supported, maintained versions of
  its core runtime and storage dependencies at initial release.

### Key Entities *(include if feature involves data)*

- **Event**: A signed Nostr event with author, kind, timestamp, tags, and
  content.
- **DeletedEvent**: A record that marks an event as deleted with reason and
  time.
- **Group**: Group metadata including visibility flags and geohash.
- **GroupRole**: A named role within a group and its permissions.
- **GroupMember**: A user's membership in a group with role history.
- **GroupBan**: A ban record for a user in a group with optional expiration.
- **GroupInvite**: An invite code and usage limits for a group.
- **GroupJoinRequest**: A pending request to join a group.
- **GroupEvent**: A mapping of group-related events to a group for queries.

## Assumptions

- The relay follows standard Nostr event validation and filtering behavior.
- Group-related events follow the NIP-29 model for metadata, membership, and
  moderation actions.
- Federation compatibility is preserved by avoiding relay-specific protocol
  extensions in client-visible behavior.

## Clarifications

### Session 2026-02-05

- Q: Who is allowed to modify group metadata, roles, membership, bans, and invites? → A: Only group owner/admin roles.
- Q: When does a join request require approval? → A: is_vetted means join requires approval.
- Q: How are conflicting group updates resolved? → A: Last-write-wins by created_at.
- Q: How should deletions affect group projections? → A: Remove from active projections but keep as deleted.
- Q: What rate limiting policy should apply? → A: Burst-then-sustained per pubkey.
- Q: What availability target should apply? → A: 99.5% monthly availability.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% of valid publish operations complete in under 2 seconds as
  experienced by clients.
- **SC-002**: 95% of standard event queries return results in under 2 seconds.
- **SC-003**: Group state projections reflect the latest valid update within
  5 seconds of event submission in 95% of cases.
- **SC-004**: At least 90% of test participants can complete a basic publish,
  query, and delete flow without assistance.
- **SC-005**: Service maintains at least 99.5% monthly availability.
