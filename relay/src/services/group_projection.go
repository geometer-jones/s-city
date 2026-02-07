package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/nbd-wtf/go-nostr"

	"s-city/src/lib"
	"s-city/src/models"
	"s-city/src/storage"
)

// GroupProjectionService applies group-related events into queryable projection tables.
type GroupProjectionService struct {
	repo         *storage.GroupRepo
	eventsRepo   *storage.EventsRepo
	relayPubKey  string
	relayPrivKey string
	vetting      *GroupVettingService
	metrics      *lib.Metrics
}

func NewGroupProjectionService(
	repo *storage.GroupRepo,
	eventsRepo *storage.EventsRepo,
	relayPubKey string,
	relayPrivKey string,
	vetting *GroupVettingService,
	metrics *lib.Metrics,
) *GroupProjectionService {
	return &GroupProjectionService{
		repo:         repo,
		eventsRepo:   eventsRepo,
		relayPubKey:  relayPubKey,
		relayPrivKey: relayPrivKey,
		vetting:      vetting,
		metrics:      metrics,
	}
}

func (s *GroupProjectionService) ApplyEvent(ctx context.Context, event models.Event) error {
	groupID := firstTagValue(event.Tags, "h")
	if groupID == "" && relayOnlyKind(event.Kind) {
		groupID = firstTagValue(event.Tags, "d")
	}
	if groupID == "" {
		return nil
	}

	membershipChanged := false
	adminsChanged := false

	switch event.Kind {
	case 9007:
		isPrivate, _ := tagBoolValue(event.Tags, "private")
		isRestricted, _ := tagBoolValue(event.Tags, "restricted")
		isVetted, _ := tagBoolValue(event.Tags, "vetted")
		isHidden, _ := tagBoolValue(event.Tags, "hidden")
		isClosed, _ := tagBoolValue(event.Tags, "closed")

		group := models.Group{
			GroupID:      groupID,
			Name:         firstTagValue(event.Tags, "name"),
			About:        firstTagValue(event.Tags, "about"),
			Picture:      firstTagValue(event.Tags, "picture"),
			Geohash:      truncateGeohash(firstTagValue(event.Tags, "g")),
			IsPrivate:    isPrivate,
			IsRestricted: isRestricted,
			IsVetted:     isVetted,
			IsHidden:     isHidden,
			IsClosed:     isClosed,
			CreatedAt:    event.CreatedAt,
			CreatedBy:    event.PubKey,
			UpdatedAt:    event.CreatedAt,
			UpdatedBy:    event.PubKey,
		}
		if err := s.repo.UpsertGroup(ctx, group); err != nil {
			return err
		}
		if err := s.repo.UpsertRole(ctx, models.GroupRole{
			GroupID:     groupID,
			RoleName:    "owner",
			Description: "Group owner",
			Permissions: ownerRolePermissions(),
			CreatedAt:   event.CreatedAt,
			CreatedBy:   event.PubKey,
			UpdatedAt:   event.CreatedAt,
			UpdatedBy:   event.PubKey,
		}); err != nil {
			return err
		}
		if err := s.repo.UpsertMember(ctx, models.GroupMember{
			GroupID:  groupID,
			PubKey:   event.PubKey,
			AddedAt:  event.CreatedAt,
			AddedBy:  event.PubKey,
			RoleName: "owner",
		}); err != nil {
			return err
		}
		membershipChanged = true
		adminsChanged = true

	case 9002:
		existing, err := s.repo.GetGroup(ctx, groupID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err == nil {
			if err := s.requirePermission(ctx, groupID, event.PubKey, models.PermissionEditMetadata); err != nil {
				return err
			}
		} else {
			existing = models.Group{GroupID: groupID, CreatedAt: event.CreatedAt, CreatedBy: event.PubKey}
		}

		if v := firstTagValue(event.Tags, "name"); v != "" {
			existing.Name = v
		}
		if v := firstTagValue(event.Tags, "about"); v != "" {
			existing.About = v
		}
		if v := firstTagValue(event.Tags, "picture"); v != "" {
			existing.Picture = v
		}
		if v := firstTagValue(event.Tags, "g"); v != "" {
			existing.Geohash = truncateGeohash(v)
		}
		if v, ok := tagBoolValue(event.Tags, "private"); ok {
			existing.IsPrivate = v
		}
		if v, ok := tagBoolValue(event.Tags, "restricted"); ok {
			existing.IsRestricted = v
		}
		if v, ok := tagBoolValue(event.Tags, "vetted"); ok {
			existing.IsVetted = v
		}
		if v, ok := tagBoolValue(event.Tags, "hidden"); ok {
			existing.IsHidden = v
		}
		if v, ok := tagBoolValue(event.Tags, "closed"); ok {
			existing.IsClosed = v
		}
		existing.UpdatedAt = event.CreatedAt
		existing.UpdatedBy = event.PubKey
		if existing.CreatedAt == 0 {
			existing.CreatedAt = event.CreatedAt
			existing.CreatedBy = event.PubKey
		}
		if err := s.repo.UpsertGroup(ctx, existing); err != nil {
			return err
		}

	case 9003:
		if err := s.requirePermission(ctx, groupID, event.PubKey, models.PermissionCreateRole); err != nil {
			return err
		}
		roleName := firstTagValue(event.Tags, "role")
		if roleName == "" {
			roleName = firstTagValue(event.Tags, "d")
		}
		if roleName == "" {
			return fmt.Errorf("role update missing role tag")
		}
		permissions := parseCSVTag(firstTagValue(event.Tags, "permissions"))
		if len(permissions) == 0 {
			permissions = parseCSVTag(firstTagValue(event.Tags, "perm"))
		}
		if err := s.repo.UpsertRole(ctx, models.GroupRole{
			GroupID:     groupID,
			RoleName:    roleName,
			Description: firstTagValue(event.Tags, "description"),
			Permissions: permissions,
			CreatedAt:   event.CreatedAt,
			CreatedBy:   event.PubKey,
			UpdatedAt:   event.CreatedAt,
			UpdatedBy:   event.PubKey,
		}); err != nil {
			return err
		}
	case 9004:
		if err := s.requirePermission(ctx, groupID, event.PubKey, models.PermissionDeleteRole); err != nil {
			return err
		}
		roleName := firstTagValue(event.Tags, "role")
		if roleName == "" {
			roleName = firstTagValue(event.Tags, "d")
		}
		if roleName == "" {
			return fmt.Errorf("delete-role missing role tag")
		}
		if err := s.repo.DeleteRole(ctx, groupID, roleName); err != nil {
			return err
		}

	case 9000:
		memberKey, requestedRole, err := parsePutUserTag(event.Tags)
		if err != nil {
			return err
		}
		if requestedRole == "" {
			requestedRole = strings.TrimSpace(firstTagValue(event.Tags, "role"))
		}
		if requestedRole == "" {
			requestedRole = "member"
		}

		previousRole, memberExists, err := s.repo.GetMemberRole(ctx, groupID, memberKey)
		if err != nil {
			return err
		}

		requiredPermission := models.PermissionAddUser
		if memberExists {
			requiredPermission = models.PermissionPromoteUser
		}
		if err := s.requirePermission(ctx, groupID, event.PubKey, requiredPermission); err != nil {
			return err
		}
		if requestedRole != "member" {
			if err := s.requirePermission(ctx, groupID, event.PubKey, models.PermissionPromoteUser); err != nil {
				return err
			}
		}
		adminRoleChanged, err := s.adminAssignmentChangedForPutUser(ctx, groupID, previousRole, requestedRole)
		if err != nil {
			return err
		}

		if err := s.repo.UpsertMember(ctx, models.GroupMember{
			GroupID:    groupID,
			PubKey:     memberKey,
			AddedAt:    event.CreatedAt,
			AddedBy:    event.PubKey,
			RoleName:   requestedRole,
			PromotedAt: event.CreatedAt,
			PromotedBy: event.PubKey,
		}); err != nil {
			return err
		}
		if err := s.repo.DeleteJoinRequest(ctx, groupID, memberKey); err != nil {
			return err
		}
		membershipChanged = true
		adminsChanged = adminRoleChanged

	case 9001:
		if err := s.requirePermission(ctx, groupID, event.PubKey, models.PermissionRemoveUser); err != nil {
			return err
		}
		memberKey := firstTagValue(event.Tags, "p")
		if memberKey == "" {
			return fmt.Errorf("remove-user missing p tag")
		}
		if err := s.repo.RemoveMember(ctx, groupID, memberKey); err != nil {
			return err
		}
		membershipChanged = true
		if hasTag(event.Tags, "ban") {
			reason := strings.TrimSpace(firstTagValue(event.Tags, "reason"))
			if reason == "" {
				reason = strings.TrimSpace(firstTagValue(event.Tags, "ban"))
			}
			if err := s.repo.UpsertBan(ctx, models.GroupBan{
				GroupID:   groupID,
				PubKey:    memberKey,
				Reason:    reason,
				BannedAt:  event.CreatedAt,
				BannedBy:  event.PubKey,
				ExpiresAt: parseInt64Tag(firstTagValue(event.Tags, "expires_at")),
			}); err != nil {
				return err
			}
		}

	case 9009:
		if err := s.requirePermission(ctx, groupID, event.PubKey, models.PermissionCreateInvite); err != nil {
			return err
		}
		code := firstTagValue(event.Tags, "code")
		if code == "" {
			code = firstTagValue(event.Tags, "invite")
		}
		if code == "" {
			return fmt.Errorf("invite event missing code")
		}
		if err := s.repo.UpsertInvite(ctx, models.GroupInvite{
			GroupID:       groupID,
			Code:          code,
			ExpiresAt:     parseInt64Tag(firstTagValue(event.Tags, "expires_at")),
			MaxUsageCount: int(parseInt64Tag(firstTagValue(event.Tags, "max_usage_count"))),
			UsageCount:    int(parseInt64Tag(firstTagValue(event.Tags, "usage_count"))),
			CreatedAt:     event.CreatedAt,
			CreatedBy:     event.PubKey,
		}); err != nil {
			return err
		}

	case 9021:
		requestKey, err := joinRequestPubKey(event)
		if err != nil {
			return err
		}
		isMember, err := s.repo.IsMember(ctx, groupID, requestKey)
		if err != nil {
			return err
		}
		if isMember {
			return fmt.Errorf("duplicate: user already member")
		}

		isBanned, err := s.repo.IsBanned(ctx, groupID, requestKey)
		if err != nil {
			return err
		}
		if isBanned {
			return fmt.Errorf("user is banned")
		}

		autoApprove, err := s.vetting.CanAutoApprove(ctx, groupID, requestKey)
		if err != nil {
			return err
		}
		if autoApprove {
			if err := s.repo.UpsertMember(ctx, models.GroupMember{
				GroupID:  groupID,
				PubKey:   requestKey,
				AddedAt:  event.CreatedAt,
				AddedBy:  event.PubKey,
				RoleName: "member",
			}); err != nil {
				return err
			}
			membershipChanged = true
		} else {
			if err := s.repo.UpsertJoinRequest(ctx, models.GroupJoinRequest{
				GroupID:   groupID,
				PubKey:    requestKey,
				CreatedAt: event.CreatedAt,
			}); err != nil {
				return err
			}
		}

	case 9022:
		if err := s.repo.RemoveMember(ctx, groupID, event.PubKey); err != nil {
			return err
		}
		if err := s.repo.DeleteJoinRequest(ctx, groupID, event.PubKey); err != nil {
			return err
		}
		membershipChanged = true

	case 9008:
		if err := s.requirePermission(ctx, groupID, event.PubKey, models.PermissionDeleteGroup); err != nil {
			return err
		}
		if err := s.repo.CloseGroup(ctx, groupID, event.CreatedAt, event.PubKey); err != nil {
			return err
		}

	case 9005:
		if err := s.requirePermission(ctx, groupID, event.PubKey, models.PermissionDeleteEvent); err != nil {
			return err
		}
		eventID := strings.TrimSpace(firstTagValue(event.Tags, "e"))
		if eventID != "" {
			if err := s.repo.RemoveGroupEventByEventID(ctx, eventID); err != nil {
				return err
			}
			if s.eventsRepo != nil {
				_, err := s.eventsRepo.GetEvent(ctx, eventID)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					return err
				}
				if err == nil {
					reason := strings.TrimSpace(firstTagValue(event.Tags, "reason"))
					if reason == "" {
						reason = "group moderation delete"
					}
					if err := s.eventsRepo.MarkDeleted(ctx, models.DeletedEvent{
						EventID:   eventID,
						DeletedAt: event.CreatedAt,
						DeletedBy: event.PubKey,
						Reason:    reason,
					}); err != nil {
						return err
					}
				}
			}
		}
	}

	if err := s.syncCanonicalStateEvents(ctx, event, groupID, membershipChanged, adminsChanged); err != nil {
		return err
	}

	if err := s.repo.AddGroupEvent(ctx, models.GroupEvent{GroupID: groupID, EventID: event.ID, CreatedAt: event.CreatedAt}); err != nil {
		return err
	}
	s.metrics.Inc("group_projection_applied_total")
	return nil
}

func (s *GroupProjectionService) ApproveJoinRequest(ctx context.Context, groupID, pubKey, approvedBy string, approvedAt int64) error {
	if err := s.requirePermission(ctx, groupID, approvedBy, models.PermissionAddUser); err != nil {
		return err
	}
	if err := s.repo.UpsertMember(ctx, models.GroupMember{
		GroupID:    groupID,
		PubKey:     pubKey,
		AddedAt:    approvedAt,
		AddedBy:    approvedBy,
		RoleName:   "member",
		PromotedAt: approvedAt,
		PromotedBy: approvedBy,
	}); err != nil {
		return err
	}
	if err := s.repo.DeleteJoinRequest(ctx, groupID, pubKey); err != nil {
		return err
	}
	if err := s.emitMembersStateEvent(ctx, groupID, approvedAt); err != nil {
		return err
	}
	s.metrics.Inc("group_join_approved_total")
	return nil
}

func (s *GroupProjectionService) ApplyDeletion(ctx context.Context, eventID string) error {
	if err := s.repo.RemoveGroupEventByEventID(ctx, eventID); err != nil {
		return err
	}
	s.metrics.Inc("group_projection_deletion_applied_total")
	return nil
}

func (s *GroupProjectionService) requirePermission(ctx context.Context, groupID, pubKey, permission string) error {
	hasPermission, err := s.repo.HasPermission(ctx, groupID, pubKey, permission)
	if err != nil {
		return err
	}
	if !hasPermission {
		if strings.TrimSpace(permission) == "" {
			return fmt.Errorf("not authorized")
		}
		return fmt.Errorf("not authorized: missing %s permission", permission)
	}
	return nil
}

func firstTagValue(tags [][]string, tagName string) string {
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == tagName {
			return tag[1]
		}
	}
	return ""
}

func hasTag(tags [][]string, tagName string) bool {
	for _, tag := range tags {
		if len(tag) >= 1 && tag[0] == tagName {
			return true
		}
	}
	return false
}

func parsePutUserTag(tags [][]string) (string, string, error) {
	for _, tag := range tags {
		if len(tag) < 2 || tag[0] != "p" {
			continue
		}

		pubKey := strings.TrimSpace(tag[1])
		if pubKey == "" {
			return "", "", fmt.Errorf("put-user missing p tag")
		}

		role := ""
		if len(tag) >= 3 {
			role = strings.TrimSpace(tag[2])
		}
		return pubKey, role, nil
	}

	return "", "", fmt.Errorf("put-user missing p tag")
}

func joinRequestPubKey(event models.Event) (string, error) {
	requestKey := strings.TrimSpace(firstTagValue(event.Tags, "p"))
	if requestKey == "" {
		return strings.TrimSpace(event.PubKey), nil
	}
	if !strings.EqualFold(requestKey, strings.TrimSpace(event.PubKey)) {
		return "", fmt.Errorf("join-request p tag must match event pubkey")
	}
	return requestKey, nil
}

func tagBoolValue(tags [][]string, tagName string) (bool, bool) {
	for _, tag := range tags {
		if len(tag) < 1 || tag[0] != tagName {
			continue
		}
		if len(tag) < 2 {
			return true, true
		}
		raw := strings.TrimSpace(tag[1])
		if raw == "" {
			return true, true
		}
		return parseBoolTag(raw), true
	}
	return false, false
}

func parseBoolTag(raw string) bool {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	switch normalized {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseInt64Tag(raw string) int64 {
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseCSVTag(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func truncateGeohash(gh string) string {
	if len(gh) <= 6 {
		return gh
	}
	return gh[:6]
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func ownerRolePermissions() []string {
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
}

func (s *GroupProjectionService) syncCanonicalStateEvents(ctx context.Context, source models.Event, groupID string, membershipChanged, adminsChanged bool) error {
	if s.eventsRepo == nil || strings.TrimSpace(s.relayPubKey) == "" || strings.TrimSpace(s.relayPrivKey) == "" {
		return nil
	}

	for _, kind := range canonicalStateKindsForSource(source.Kind, membershipChanged, adminsChanged) {
		var err error
		switch kind {
		case 39000:
			err = s.emitGroupMetadataStateEvent(ctx, groupID, source.CreatedAt)
		case 39001:
			err = s.emitAdminsStateEvent(ctx, groupID, source.CreatedAt)
		case 39002:
			err = s.emitMembersStateEvent(ctx, groupID, source.CreatedAt)
		case 39003:
			err = s.emitRolesStateEvent(ctx, groupID, source.CreatedAt)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func canonicalStateKindsForSource(kind int, membershipChanged, adminsChanged bool) []int {
	kinds := make([]int, 0, 4)
	appendUnique := func(eventKind int) {
		for _, existing := range kinds {
			if existing == eventKind {
				return
			}
		}
		kinds = append(kinds, eventKind)
	}

	switch kind {
	case 9007:
		appendUnique(39000)
		appendUnique(39002)
		appendUnique(39003)
		appendUnique(39001)
	case 9002, 9008:
		appendUnique(39000)
	case 9003, 9004:
		appendUnique(39003)
	case 9000, 9001, 9022:
		if membershipChanged {
			appendUnique(39002)
		}
	case 9021:
		if membershipChanged {
			appendUnique(39002)
		}
	}

	if kind == 9000 && adminsChanged {
		appendUnique(39001)
	}

	return kinds
}

func (s *GroupProjectionService) emitGroupMetadataStateEvent(ctx context.Context, groupID string, createdAt int64) error {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	return s.upsertCanonicalStateEvent(ctx, 39000, groupID, createdAt, groupMetadataStateTags(group))
}

func (s *GroupProjectionService) emitMembersStateEvent(ctx context.Context, groupID string, createdAt int64) error {
	members, err := s.repo.ListMembers(ctx, groupID)
	if err != nil {
		return err
	}

	return s.upsertCanonicalStateEvent(ctx, 39002, groupID, createdAt, groupMembersStateTags(groupID, members))
}

func (s *GroupProjectionService) emitAdminsStateEvent(ctx context.Context, groupID string, createdAt int64) error {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	members, err := s.repo.ListMembers(ctx, groupID)
	if err != nil {
		return err
	}
	rolePermissionsByRole, err := s.rolePermissionsByName(ctx, groupID)
	if err != nil {
		return err
	}

	adminRolesByPubKey := make(map[string]string)
	for _, member := range members {
		roleName := defaultString(member.RoleName, "member")
		normalizedRole := normalizeRoleName(roleName)
		if roleGrantsAdmin(normalizedRole, rolePermissionsByRole[normalizedRole]) {
			adminRolesByPubKey[member.PubKey] = roleName
		}
	}
	if group.CreatedBy != "" {
		if _, exists := adminRolesByPubKey[group.CreatedBy]; !exists {
			adminRolesByPubKey[group.CreatedBy] = "owner"
		}
	}

	pubKeys := make([]string, 0, len(adminRolesByPubKey))
	for pubKey := range adminRolesByPubKey {
		pubKeys = append(pubKeys, pubKey)
	}
	sort.Strings(pubKeys)

	tags := [][]string{{"d", groupID}}
	for _, pubKey := range pubKeys {
		roleName := defaultString(adminRolesByPubKey[pubKey], "owner")
		tags = append(tags, []string{"p", pubKey, roleName})
	}

	return s.upsertCanonicalStateEvent(ctx, 39001, groupID, createdAt, tags)
}

func (s *GroupProjectionService) emitRolesStateEvent(ctx context.Context, groupID string, createdAt int64) error {
	roles, err := s.repo.ListRoles(ctx, groupID)
	if err != nil {
		return err
	}

	return s.upsertCanonicalStateEvent(ctx, 39003, groupID, createdAt, groupRolesStateTags(groupID, roles))
}

func groupMetadataStateTags(group models.Group) [][]string {
	tags := [][]string{{"d", group.GroupID}}
	if group.Name != "" {
		tags = append(tags, []string{"name", group.Name})
	}
	if group.Picture != "" {
		tags = append(tags, []string{"picture", group.Picture})
	}
	if group.About != "" {
		tags = append(tags, []string{"about", group.About})
	}
	if group.Geohash != "" {
		tags = append(tags, []string{"g", truncateGeohash(group.Geohash)})
	}
	if group.IsPrivate {
		tags = append(tags, []string{"private"})
	}
	if group.IsRestricted {
		tags = append(tags, []string{"restricted"})
	}
	// Keep vetted as an extension tag but encode as presence-style.
	if group.IsVetted {
		tags = append(tags, []string{"vetted"})
	}
	if group.IsHidden {
		tags = append(tags, []string{"hidden"})
	}
	if group.IsClosed {
		tags = append(tags, []string{"closed"})
	}
	return tags
}

func groupMembersStateTags(groupID string, members []models.GroupMember) [][]string {
	tags := [][]string{{"d", groupID}}
	for _, member := range members {
		tags = append(tags, []string{"p", member.PubKey})
	}
	return tags
}

func groupRolesStateTags(groupID string, roles []models.GroupRole) [][]string {
	tags := [][]string{{"d", groupID}}
	for _, role := range roles {
		roleTag := []string{"role", role.RoleName}
		if role.Description != "" {
			roleTag = append(roleTag, role.Description)
		}
		tags = append(tags, roleTag)
	}
	return tags
}

func (s *GroupProjectionService) upsertCanonicalStateEvent(ctx context.Context, kind int, groupID string, createdAt int64, tags [][]string) error {
	nostrTags := make(nostr.Tags, 0, len(tags))
	for _, tag := range tags {
		nostrTag := make(nostr.Tag, len(tag))
		copy(nostrTag, tag)
		nostrTags = append(nostrTags, nostrTag)
	}

	nostrEvent := nostr.Event{
		CreatedAt: nostr.Timestamp(createdAt),
		Kind:      kind,
		Tags:      nostrTags,
		Content:   "",
	}
	if err := nostrEvent.Sign(s.relayPrivKey); err != nil {
		return fmt.Errorf("sign canonical state event kind %d: %w", kind, err)
	}
	if !strings.EqualFold(nostrEvent.PubKey, s.relayPubKey) {
		return fmt.Errorf("signed canonical state event pubkey does not match relay pubkey")
	}

	event := models.Event{
		ID:        nostrEvent.ID,
		PubKey:    strings.ToLower(nostrEvent.PubKey),
		CreatedAt: createdAt,
		Kind:      kind,
		Tags:      tags,
		Content:   "",
		Sig:       nostrEvent.Sig,
	}
	if err := s.eventsRepo.UpsertParameterizedReplaceableEvent(ctx, event, groupID); err != nil {
		return err
	}
	if err := s.repo.AddGroupEvent(ctx, models.GroupEvent{
		GroupID:   groupID,
		EventID:   event.ID,
		CreatedAt: createdAt,
	}); err != nil {
		return err
	}
	return nil
}

func (s *GroupProjectionService) adminAssignmentChangedForPutUser(ctx context.Context, groupID, previousRole, requestedRole string) (bool, error) {
	rolePermissionsByRole, err := s.rolePermissionsByName(ctx, groupID)
	if err != nil {
		return false, err
	}
	return adminAssignmentChanged(previousRole, requestedRole, rolePermissionsByRole), nil
}

func (s *GroupProjectionService) rolePermissionsByName(ctx context.Context, groupID string) (map[string][]string, error) {
	roles, err := s.repo.ListRoles(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string, len(roles))
	for _, role := range roles {
		roleName := normalizeRoleName(role.RoleName)
		permissions := make([]string, 0, len(role.Permissions))
		permissions = append(permissions, role.Permissions...)
		out[roleName] = permissions
	}
	return out, nil
}

func adminAssignmentChanged(previousRole, requestedRole string, rolePermissionsByRole map[string][]string) bool {
	previousRole = normalizeRoleName(previousRole)
	requestedRole = normalizeRoleName(requestedRole)
	if previousRole == requestedRole {
		return false
	}

	previousWasAdmin := roleGrantsAdmin(previousRole, rolePermissionsByRole[previousRole])
	requestedIsAdmin := roleGrantsAdmin(requestedRole, rolePermissionsByRole[requestedRole])
	if previousWasAdmin != requestedIsAdmin {
		return true
	}
	return previousWasAdmin && requestedIsAdmin
}

func roleGrantsAdmin(roleName string, permissions []string) bool {
	roleName = normalizeRoleName(roleName)
	switch roleName {
	case "":
		return false
	case "owner", "admin":
		return true
	}
	for _, permission := range permissions {
		if normalizePermission(permission) == models.PermissionAdmin {
			return true
		}
	}
	return false
}

func normalizeRoleName(roleName string) string {
	return strings.TrimSpace(strings.ToLower(roleName))
}

func normalizePermission(permission string) string {
	return strings.TrimSpace(strings.ToLower(permission))
}
