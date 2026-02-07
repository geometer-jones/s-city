# Data Model: Nostr Relay with Group Projections

## Entities

### Event
- **Fields**: id (hex), pubkey (hex), created_at (unix seconds), kind (int),
  tags (json array), content (string), sig (hex)
- **Relationships**: 1:N to EventTag; 1:1 to DeletedEvent
- **Notes**: id is unique; tags include NIP-01 and NIP-29 tags.

### EventTag
- **Fields**: id (serial), event_id (FK), tag_index (int), tag_name (string),
  tag_value (string), tag_array (json array)
- **Relationships**: N:1 to Event
- **Notes**: unique on (event_id, tag_index) for deterministic projection.

### DeletedEvent
- **Fields**: event_id (FK), deleted_at (unix seconds), deleted_by (pubkey),
  reason (string)
- **Relationships**: 1:1 to Event
- **Notes**: deletion does not remove the underlying event row.

### Group
- **Fields**: group_id (string), name, about, picture, geohash, is_private,
  is_restricted, is_vetted, is_hidden, is_closed, created_at, created_by,
  updated_at, updated_by
- **Relationships**: 1:N to GroupRole, GroupMember, GroupBan, GroupInvite,
  GroupJoinRequest, GroupEvent
- **Notes**: `is_vetted` means join requires approval.

### GroupRole
- **Fields**: group_id (FK), role_name, description, permissions (string[]),
  created_at, created_by, updated_at, updated_by
- **Relationships**: N:1 to Group
- **Notes**: composite PK (group_id, role_name).

### GroupMember
- **Fields**: group_id (FK), pubkey, added_at, added_by, role_name,
  promoted_at, promoted_by
- **Relationships**: N:1 to Group; optional reference to GroupRole by
  (group_id, role_name)
- **Notes**: composite PK (group_id, pubkey).

### GroupBan
- **Fields**: group_id (FK), pubkey, reason, banned_at, banned_by, expires_at
- **Relationships**: N:1 to Group
- **Notes**: composite PK (group_id, pubkey).

### GroupInvite
- **Fields**: group_id (FK), code, expires_at, max_usage_count, usage_count,
  created_at, created_by
- **Relationships**: N:1 to Group
- **Notes**: composite PK (group_id, code); code unique.

### GroupJoinRequest
- **Fields**: group_id (FK), pubkey, created_at
- **Relationships**: N:1 to Group
- **Notes**: composite PK (group_id, pubkey).

### GroupEvent
- **Fields**: group_id (FK), event_id (FK), created_at
- **Relationships**: N:1 to Group; N:1 to Event
- **Notes**: composite PK (group_id, event_id).

## Relationships

- Event 1:N EventTag
- Event 1:1 DeletedEvent
- Group 1:N GroupRole, GroupMember, GroupBan, GroupInvite, GroupJoinRequest,
  GroupEvent
- GroupEvent N:1 Event

## Validation Rules

- Event id and pubkey MUST be valid hex and match signature.
- tag_index is 0-based and unique per event.
- Geohash precision MUST be <= level 6; reject higher precision.
- For vetted groups, join requests require admin approval before membership.
- Conflicting group updates resolve via last-write-wins by created_at.

## State Transitions

- **Group metadata**: create -> update -> delete (if deleted via events, mark
  hidden/closed rather than hard delete).
- **Membership**: request -> approved -> member; banned -> expired/unbanned.
- **Invites**: active -> used (increment usage) -> expired.
- **Event deletion**: active -> deleted (remains queryable as deleted record).
