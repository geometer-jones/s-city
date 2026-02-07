---

description: "Task list template for feature implementation"
---

# Tasks: Nostr Relay with Group Projections

**Input**: Design documents from `/specs/001-nostr-relay-groups/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are OPTIONAL - no test tasks included because the spec did not request TDD.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `src/`, `tests/` at repository root

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Create project skeleton in `src/` and `tests/` directories
- [x] T002 Initialize Go module in `go.mod`
- [x] T003 Add configuration loader in `src/lib/config.go`
- [x] T004 Add logging setup in `src/lib/logging.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Create initial database schema migration in `src/storage/migrations/001_init.sql`
- [x] T006 Add database connection pool in `src/storage/db.go`
- [x] T007 Implement event storage repository in `src/storage/events_repo.go`
- [x] T008 Implement event tag normalization logic in `src/storage/event_tags_repo.go`
- [x] T009 Implement signature validation in `src/services/validation.go`
- [x] T010 Implement PoW and rate limiting controls in `src/services/abuse_controls.go`
- [x] T011 Implement relay server bootstrap in `src/relay/server.go`
- [x] T012 Implement Nostr protocol handler wiring in `src/relay/nostr_handler.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Publish and Retrieve Standard Events (Priority: P1) 🎯 MVP

**Goal**: Accept, validate, store, and query standard Nostr events.

**Independent Test**: Publish a valid event and query it by author/kind/tags; invalid events are rejected.

### Implementation for User Story 1

- [x] T013 [P] [US1] Define Event model in `src/models/event.go`
- [x] T014 [P] [US1] Define EventTag model in `src/models/event_tag.go`
- [x] T015 [US1] Implement event ingest service in `src/services/event_ingest.go`
- [x] T016 [US1] Implement event query service in `src/services/event_query.go`
- [x] T017 [US1] Add relay routes for event publish/query in `src/relay/event_routes.go`

**Checkpoint**: User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Manage Group State Projections (Priority: P2)

**Goal**: Project group metadata, roles, membership, bans, invites, join requests, and group events.

**Independent Test**: Submit a sequence of group events and verify projections match expected state.

### Implementation for User Story 2

- [x] T018 [P] [US2] Define Group model in `src/models/group.go`
- [x] T019 [P] [US2] Define GroupRole model in `src/models/group_role.go`
- [x] T020 [P] [US2] Define GroupMember model in `src/models/group_member.go`
- [x] T021 [P] [US2] Define GroupBan model in `src/models/group_ban.go`
- [x] T022 [P] [US2] Define GroupInvite model in `src/models/group_invite.go`
- [x] T023 [P] [US2] Define GroupJoinRequest model in `src/models/group_join_request.go`
- [x] T024 [P] [US2] Define GroupEvent model in `src/models/group_event.go`
- [x] T025 [US2] Implement group projection repository in `src/storage/group_repo.go`
- [x] T026 [US2] Implement projection updater in `src/services/group_projection.go`
- [x] T027 [US2] Implement vetted join approval rules in `src/services/group_vetting.go`
- [x] T028 [US2] Add relay routes for group queries in `src/relay/group_routes.go`

**Checkpoint**: User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Respect Deletions and Visibility (Priority: P3)

**Goal**: Process deletions and remove deleted items from active results while retaining deleted records.

**Independent Test**: Submit a delete event and verify the deleted item is excluded from active queries and projections.

### Implementation for User Story 3

- [x] T029 [P] [US3] Define DeletedEvent model in `src/models/deleted_event.go`
- [x] T030 [US3] Implement delete ingest service in `src/services/event_delete.go`
- [x] T031 [US3] Update query filtering for deletions in `src/services/event_query.go`
- [x] T032 [US3] Apply deletion effects to group projections in `src/services/group_projection.go`

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T033 Update developer quickstart in `/Users/peterwei/wokspace/s-city/specs/001-nostr-relay-groups/quickstart.md`
- [x] T034 Add metrics hooks in `src/lib/metrics.go`
- [x] T035 Constitution compliance review notes in `/Users/peterwei/wokspace/s-city/specs/001-nostr-relay-groups/plan.md`
- [x] T036 Run quickstart validation and record results in `/Users/peterwei/wokspace/s-city/specs/001-nostr-relay-groups/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May share projection services but independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Depends on shared query and projection services

### Within Each User Story

- Models before services
- Services before relay routes
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- Models within a story marked [P] can run in parallel

---

## Parallel Example: User Story 2

```bash
Task: "Define GroupRole model in src/models/group_role.go"
Task: "Define GroupMember model in src/models/group_member.go"
Task: "Define GroupBan model in src/models/group_ban.go"
Task: "Define GroupInvite model in src/models/group_invite.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready
