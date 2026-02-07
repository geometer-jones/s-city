package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"s-city/src/models"
)

type GroupFilter struct {
	GeohashPrefix string
	IsPrivate     *bool
	IsVetted      *bool
	UpdatedSince  *int64
	Limit         int
}

type GroupRepo struct {
	pool *pgxpool.Pool
}

func NewGroupRepo(pool *pgxpool.Pool) *GroupRepo {
	return &GroupRepo{pool: pool}
}

func (r *GroupRepo) UpsertGroup(ctx context.Context, group models.Group) error {
	if group.Geohash != "" && len(group.Geohash) > 6 {
		return fmt.Errorf("geohash precision exceeds level 6")
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO groups (
			group_id, name, about, picture, geohash, is_private, is_restricted,
			is_vetted, is_hidden, is_closed, created_at, created_by, updated_at, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (group_id) DO UPDATE
		SET name = EXCLUDED.name,
			about = EXCLUDED.about,
			picture = EXCLUDED.picture,
			geohash = EXCLUDED.geohash,
			is_private = EXCLUDED.is_private,
			is_restricted = EXCLUDED.is_restricted,
			is_vetted = EXCLUDED.is_vetted,
			is_hidden = EXCLUDED.is_hidden,
			is_closed = EXCLUDED.is_closed,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
		WHERE EXCLUDED.updated_at >= groups.updated_at
	`,
		group.GroupID, group.Name, group.About, group.Picture, group.Geohash,
		group.IsPrivate, group.IsRestricted, group.IsVetted, group.IsHidden, group.IsClosed,
		group.CreatedAt, group.CreatedBy, group.UpdatedAt, group.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("upsert group: %w", err)
	}
	return nil
}

func (r *GroupRepo) CloseGroup(ctx context.Context, groupID string, updatedAt int64, updatedBy string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE groups
		SET is_hidden = TRUE,
			is_closed = TRUE,
			updated_at = $2,
			updated_by = $3
		WHERE group_id = $1
	`, groupID, updatedAt, updatedBy)
	if err != nil {
		return fmt.Errorf("close group: %w", err)
	}
	return nil
}

func (r *GroupRepo) UpsertRole(ctx context.Context, role models.GroupRole) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO group_roles (
			group_id, role_name, description, permissions,
			created_at, created_by, updated_at, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (group_id, role_name) DO UPDATE
		SET description = EXCLUDED.description,
			permissions = EXCLUDED.permissions,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
		WHERE EXCLUDED.updated_at >= group_roles.updated_at
	`, role.GroupID, role.RoleName, role.Description, role.Permissions,
		role.CreatedAt, role.CreatedBy, role.UpdatedAt, role.UpdatedBy)
	if err != nil {
		return fmt.Errorf("upsert group role: %w", err)
	}
	return nil
}

func (r *GroupRepo) DeleteRole(ctx context.Context, groupID, roleName string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM group_roles
		WHERE group_id = $1 AND role_name = $2
	`, groupID, roleName)
	if err != nil {
		return fmt.Errorf("delete group role: %w", err)
	}
	return nil
}

func (r *GroupRepo) UpsertMember(ctx context.Context, member models.GroupMember) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO group_members (
			group_id, pubkey, added_at, added_by, role_name, promoted_at, promoted_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (group_id, pubkey) DO UPDATE
		SET role_name = EXCLUDED.role_name,
			promoted_at = EXCLUDED.promoted_at,
			promoted_by = EXCLUDED.promoted_by,
			added_at = EXCLUDED.added_at,
			added_by = EXCLUDED.added_by
		WHERE EXCLUDED.added_at >= group_members.added_at
	`, member.GroupID, member.PubKey, member.AddedAt, member.AddedBy,
		member.RoleName, member.PromotedAt, member.PromotedBy)
	if err != nil {
		return fmt.Errorf("upsert group member: %w", err)
	}
	return nil
}

func (r *GroupRepo) RemoveMember(ctx context.Context, groupID, pubKey string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM group_members WHERE group_id = $1 AND pubkey = $2
	`, groupID, pubKey)
	if err != nil {
		return fmt.Errorf("remove group member: %w", err)
	}
	return nil
}

func (r *GroupRepo) UpsertBan(ctx context.Context, ban models.GroupBan) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO group_bans (group_id, pubkey, reason, banned_at, banned_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (group_id, pubkey) DO UPDATE
		SET reason = EXCLUDED.reason,
			banned_at = EXCLUDED.banned_at,
			banned_by = EXCLUDED.banned_by,
			expires_at = EXCLUDED.expires_at
		WHERE EXCLUDED.banned_at >= group_bans.banned_at
	`, ban.GroupID, ban.PubKey, ban.Reason, ban.BannedAt, ban.BannedBy, ban.ExpiresAt)
	if err != nil {
		return fmt.Errorf("upsert group ban: %w", err)
	}
	return nil
}

func (r *GroupRepo) UpsertInvite(ctx context.Context, invite models.GroupInvite) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO group_invites (
			group_id, code, expires_at, max_usage_count, usage_count, created_at, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (group_id, code) DO UPDATE
		SET expires_at = EXCLUDED.expires_at,
			max_usage_count = EXCLUDED.max_usage_count,
			usage_count = EXCLUDED.usage_count
		WHERE EXCLUDED.created_at >= group_invites.created_at
	`, invite.GroupID, invite.Code, invite.ExpiresAt, invite.MaxUsageCount,
		invite.UsageCount, invite.CreatedAt, invite.CreatedBy)
	if err != nil {
		return fmt.Errorf("upsert group invite: %w", err)
	}
	return nil
}

func (r *GroupRepo) UpsertJoinRequest(ctx context.Context, req models.GroupJoinRequest) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO group_join_requests (group_id, pubkey, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (group_id, pubkey) DO UPDATE
		SET created_at = GREATEST(group_join_requests.created_at, EXCLUDED.created_at)
	`, req.GroupID, req.PubKey, req.CreatedAt)
	if err != nil {
		return fmt.Errorf("upsert join request: %w", err)
	}
	return nil
}

func (r *GroupRepo) DeleteJoinRequest(ctx context.Context, groupID, pubKey string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM group_join_requests WHERE group_id = $1 AND pubkey = $2
	`, groupID, pubKey)
	if err != nil {
		return fmt.Errorf("delete join request: %w", err)
	}
	return nil
}

func (r *GroupRepo) AddGroupEvent(ctx context.Context, ge models.GroupEvent) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO group_events (group_id, event_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (group_id, event_id) DO UPDATE
		SET created_at = GREATEST(group_events.created_at, EXCLUDED.created_at)
	`, ge.GroupID, ge.EventID, ge.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert group event mapping: %w", err)
	}
	return nil
}

func (r *GroupRepo) RemoveGroupEventByEventID(ctx context.Context, eventID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM group_events WHERE event_id = $1`, eventID)
	if err != nil {
		return fmt.Errorf("remove group event mapping: %w", err)
	}
	return nil
}

func (r *GroupRepo) GetGroup(ctx context.Context, groupID string) (models.Group, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT group_id, name, about, picture, geohash, is_private, is_restricted,
			is_vetted, is_hidden, is_closed, created_at, created_by, updated_at, updated_by
		FROM groups
		WHERE group_id = $1
	`, groupID)

	var group models.Group
	if err := row.Scan(&group.GroupID, &group.Name, &group.About, &group.Picture, &group.Geohash,
		&group.IsPrivate, &group.IsRestricted, &group.IsVetted, &group.IsHidden, &group.IsClosed,
		&group.CreatedAt, &group.CreatedBy, &group.UpdatedAt, &group.UpdatedBy); err != nil {
		return models.Group{}, err
	}
	return group, nil
}

func (r *GroupRepo) ListGroups(ctx context.Context, filter GroupFilter) ([]models.Group, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}

	var b strings.Builder
	args := make([]any, 0, 8)
	argIdx := 1

	b.WriteString(`
		SELECT group_id, name, about, picture, geohash, is_private, is_restricted,
			is_vetted, is_hidden, is_closed, created_at, created_by, updated_at, updated_by
		FROM groups
		WHERE 1=1
	`)

	if filter.GeohashPrefix != "" {
		b.WriteString(fmt.Sprintf("AND geohash LIKE $%d || '%%'\n", argIdx))
		args = append(args, filter.GeohashPrefix)
		argIdx++
	}
	if filter.IsPrivate != nil {
		b.WriteString(fmt.Sprintf("AND is_private = $%d\n", argIdx))
		args = append(args, *filter.IsPrivate)
		argIdx++
	}
	if filter.IsVetted != nil {
		b.WriteString(fmt.Sprintf("AND is_vetted = $%d\n", argIdx))
		args = append(args, *filter.IsVetted)
		argIdx++
	}
	if filter.UpdatedSince != nil {
		b.WriteString(fmt.Sprintf("AND updated_at >= $%d\n", argIdx))
		args = append(args, *filter.UpdatedSince)
		argIdx++
	}

	b.WriteString("ORDER BY updated_at DESC\n")
	b.WriteString(fmt.Sprintf("LIMIT $%d", argIdx))
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query groups: %w", err)
	}
	defer rows.Close()

	groups := make([]models.Group, 0)
	for rows.Next() {
		var group models.Group
		if err := rows.Scan(&group.GroupID, &group.Name, &group.About, &group.Picture, &group.Geohash,
			&group.IsPrivate, &group.IsRestricted, &group.IsVetted, &group.IsHidden, &group.IsClosed,
			&group.CreatedAt, &group.CreatedBy, &group.UpdatedAt, &group.UpdatedBy); err != nil {
			return nil, fmt.Errorf("scan group row: %w", err)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group rows: %w", err)
	}
	return groups, nil
}

func (r *GroupRepo) ListMembers(ctx context.Context, groupID string) ([]models.GroupMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT group_id, pubkey, added_at, added_by, role_name, promoted_at, promoted_by
		FROM group_members
		WHERE group_id = $1
		ORDER BY added_at ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("query group members: %w", err)
	}
	defer rows.Close()

	members := make([]models.GroupMember, 0)
	for rows.Next() {
		var member models.GroupMember
		if err := rows.Scan(&member.GroupID, &member.PubKey, &member.AddedAt, &member.AddedBy,
			&member.RoleName, &member.PromotedAt, &member.PromotedBy); err != nil {
			return nil, fmt.Errorf("scan group member row: %w", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group members: %w", err)
	}
	return members, nil
}

func (r *GroupRepo) IsMember(ctx context.Context, groupID, pubKey string) (bool, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM group_members
			WHERE group_id = $1 AND pubkey = $2
		)
	`, groupID, pubKey)

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("scan member existence: %w", err)
	}
	return exists, nil
}

func (r *GroupRepo) GetMemberRole(ctx context.Context, groupID, pubKey string) (string, bool, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT role_name
		FROM group_members
		WHERE group_id = $1 AND pubkey = $2
	`, groupID, pubKey)

	var roleName string
	if err := row.Scan(&roleName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("scan member role: %w", err)
	}
	return roleName, true, nil
}

func (r *GroupRepo) ListRoles(ctx context.Context, groupID string) ([]models.GroupRole, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT group_id, role_name, description, permissions, created_at, created_by, updated_at, updated_by
		FROM group_roles
		WHERE group_id = $1
		ORDER BY role_name ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("query group roles: %w", err)
	}
	defer rows.Close()

	roles := make([]models.GroupRole, 0)
	for rows.Next() {
		var role models.GroupRole
		if err := rows.Scan(&role.GroupID, &role.RoleName, &role.Description, &role.Permissions,
			&role.CreatedAt, &role.CreatedBy, &role.UpdatedAt, &role.UpdatedBy); err != nil {
			return nil, fmt.Errorf("scan group role row: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group roles: %w", err)
	}
	return roles, nil
}

func (r *GroupRepo) ListBans(ctx context.Context, groupID string) ([]models.GroupBan, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT group_id, pubkey, reason, banned_at, banned_by, expires_at
		FROM group_bans
		WHERE group_id = $1
		ORDER BY banned_at DESC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("query group bans: %w", err)
	}
	defer rows.Close()

	bans := make([]models.GroupBan, 0)
	for rows.Next() {
		var ban models.GroupBan
		if err := rows.Scan(&ban.GroupID, &ban.PubKey, &ban.Reason, &ban.BannedAt, &ban.BannedBy, &ban.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan group ban row: %w", err)
		}
		bans = append(bans, ban)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group bans: %w", err)
	}
	return bans, nil
}

func (r *GroupRepo) ListInvites(ctx context.Context, groupID string) ([]models.GroupInvite, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT group_id, code, expires_at, max_usage_count, usage_count, created_at, created_by
		FROM group_invites
		WHERE group_id = $1
		ORDER BY created_at DESC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("query group invites: %w", err)
	}
	defer rows.Close()

	invites := make([]models.GroupInvite, 0)
	for rows.Next() {
		var invite models.GroupInvite
		if err := rows.Scan(&invite.GroupID, &invite.Code, &invite.ExpiresAt, &invite.MaxUsageCount,
			&invite.UsageCount, &invite.CreatedAt, &invite.CreatedBy); err != nil {
			return nil, fmt.Errorf("scan group invite row: %w", err)
		}
		invites = append(invites, invite)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group invites: %w", err)
	}
	return invites, nil
}

func (r *GroupRepo) HasPermission(ctx context.Context, groupID, pubKey, permission string) (bool, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT
			g.created_by,
			COALESCE(gm.role_name, ''),
			COALESCE(gr.permissions, '{}'::TEXT[])
		FROM groups g
		LEFT JOIN group_members gm
			ON gm.group_id = g.group_id AND gm.pubkey = $2
		LEFT JOIN group_roles gr
			ON gr.group_id = gm.group_id AND gr.role_name = gm.role_name
		WHERE g.group_id = $1
	`, groupID, pubKey)

	var createdBy string
	var roleName string
	permissions := make([]string, 0)
	if err := row.Scan(&createdBy, &roleName, &permissions); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("scan permission subject: %w", err)
	}

	if createdBy == pubKey {
		return true, nil
	}
	return roleHasPermission(roleName, permissions, permission), nil
}

func (r *GroupRepo) IsAdmin(ctx context.Context, groupID, pubKey string) (bool, error) {
	group, err := r.GetGroup(ctx, groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("load group: %w", err)
	}
	if group.CreatedBy == pubKey {
		return true, nil
	}

	row := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM group_members gm
			LEFT JOIN group_roles gr
				ON gr.group_id = gm.group_id AND gr.role_name = gm.role_name
			WHERE gm.group_id = $1
			  AND gm.pubkey = $2
			  AND (
				gm.role_name IN ('owner', 'admin')
				OR gr.permissions @> ARRAY['admin']::TEXT[]
			  )
		)
	`, groupID, pubKey)

	var isAdmin bool
	if err := row.Scan(&isAdmin); err != nil {
		return false, fmt.Errorf("scan admin status: %w", err)
	}
	return isAdmin, nil
}

func roleHasPermission(roleName string, rolePermissions []string, requiredPermission string) bool {
	requiredPermission = normalizePermission(requiredPermission)
	if requiredPermission == "" {
		return false
	}

	roleName = strings.TrimSpace(strings.ToLower(roleName))
	if roleName == "owner" {
		return true
	}

	allowed := make(map[string]struct{}, len(rolePermissions)+len(defaultRolePermissions(roleName)))
	for _, permission := range defaultRolePermissions(roleName) {
		normalized := normalizePermission(permission)
		if normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}
	for _, permission := range rolePermissions {
		normalized := normalizePermission(permission)
		if normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}

	if _, ok := allowed[requiredPermission]; ok {
		return true
	}
	_, hasAdmin := allowed[models.PermissionAdmin]
	return hasAdmin
}

func normalizePermission(permission string) string {
	return strings.TrimSpace(strings.ToLower(permission))
}

func defaultRolePermissions(roleName string) []string {
	switch roleName {
	case "admin":
		return []string{
			models.PermissionAddUser,
			models.PermissionPromoteUser,
			models.PermissionRemoveUser,
			models.PermissionEditMetadata,
			models.PermissionCreateRole,
			models.PermissionDeleteRole,
			models.PermissionDeleteEvent,
			models.PermissionCreateGroup,
			models.PermissionDeleteGroup,
			models.PermissionCreateInvite,
		}
	default:
		return nil
	}
}

func (r *GroupRepo) IsBanned(ctx context.Context, groupID, pubKey string) (bool, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT expires_at
		FROM group_bans
		WHERE group_id = $1 AND pubkey = $2
	`, groupID, pubKey)

	var expiresAt int64
	if err := row.Scan(&expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("scan ban status: %w", err)
	}

	if expiresAt == 0 {
		return true, nil
	}
	return expiresAt >= time.Now().Unix(), nil
}
