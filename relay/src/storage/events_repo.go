package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"s-city/src/models"
)

type EventFilter struct {
	Author         string
	Kind           *int
	Since          *int64
	Until          *int64
	UntilID        string
	Tag            string
	Limit          int
	IncludeDeleted bool
}

type EventsRepo struct {
	pool     *pgxpool.Pool
	tagsRepo *EventTagsRepo
}

func NewEventsRepo(pool *pgxpool.Pool, tagsRepo *EventTagsRepo) *EventsRepo {
	return &EventsRepo{pool: pool, tagsRepo: tagsRepo}
}

func (r *EventsRepo) InsertEvent(ctx context.Context, event models.Event) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.insertEventTx(ctx, tx, event); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// UpsertReplaceableEvent stores a replaceable event by replacing older
// events with the same (pubkey, kind).
func (r *EventsRepo) UpsertReplaceableEvent(ctx context.Context, event models.Event) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT e.id, e.created_at
		FROM events e
		WHERE e.pubkey = $1
		  AND e.kind = $2
	`, event.PubKey, event.Kind)
	if err != nil {
		return fmt.Errorf("query existing replaceable event: %w", err)
	}
	if err := r.upsertLatestEventTx(ctx, tx, event, rows); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// UpsertParameterizedReplaceableEvent stores a parameterized replaceable event
// by replacing older events with the same (pubkey, kind, d-tag value).
// A missing d-tag is treated as the empty d address.
func (r *EventsRepo) UpsertParameterizedReplaceableEvent(ctx context.Context, event models.Event, dTagValue string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT DISTINCT e.id, e.created_at
		FROM events e
		LEFT JOIN event_tags et
		  ON et.event_id = e.id
		 AND et.tag_name = 'd'
		WHERE e.pubkey = $1
		  AND e.kind = $2
		  AND (
			  ($3 = '' AND (et.event_id IS NULL OR et.tag_value = ''))
			  OR ($3 <> '' AND et.tag_value = $3)
		  )
	`, event.PubKey, event.Kind, dTagValue)
	if err != nil {
		return fmt.Errorf("query existing parameterized replaceable event: %w", err)
	}
	if err := r.upsertLatestEventTx(ctx, tx, event, rows); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *EventsRepo) upsertLatestEventTx(ctx context.Context, tx pgx.Tx, event models.Event, rows pgx.Rows) error {
	existingIDs := make([]string, 0, 2)
	bestExistingID := ""
	bestExistingCreatedAt := int64(0)
	hasExisting := false
	for rows.Next() {
		var existingID string
		var existingCreatedAt int64
		if err := rows.Scan(&existingID, &existingCreatedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scan existing replaceable event id: %w", err)
		}
		if !hasExisting || compareReplaceableVersion(existingCreatedAt, existingID, bestExistingCreatedAt, bestExistingID) > 0 {
			bestExistingID = existingID
			bestExistingCreatedAt = existingCreatedAt
			hasExisting = true
		}
		if existingID != event.ID {
			existingIDs = append(existingIDs, existingID)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate existing replaceable event ids: %w", err)
	}
	rows.Close()

	if hasExisting {
		switch compareReplaceableVersion(event.CreatedAt, event.ID, bestExistingCreatedAt, bestExistingID) {
		case -1:
			return nil
		case 0:
			return nil
		}
	}

	for _, existingID := range existingIDs {
		if _, err := tx.Exec(ctx, `DELETE FROM events WHERE id = $1`, existingID); err != nil {
			return fmt.Errorf("delete existing replaceable event %s: %w", existingID, err)
		}
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM event_tags
		WHERE event_id = $1
	`, event.ID); err != nil {
		return fmt.Errorf("delete old tags for event %s: %w", event.ID, err)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM events
		WHERE id = $1
	`, event.ID); err != nil {
		return fmt.Errorf("delete old event %s: %w", event.ID, err)
	}

	if err := r.insertEventTx(ctx, tx, event); err != nil {
		return err
	}

	return nil
}

func compareReplaceableVersion(createdAtA int64, idA string, createdAtB int64, idB string) int {
	switch {
	case createdAtA > createdAtB:
		return 1
	case createdAtA < createdAtB:
		return -1
	}

	idA = strings.ToLower(strings.TrimSpace(idA))
	idB = strings.ToLower(strings.TrimSpace(idB))
	switch strings.Compare(idA, idB) {
	case -1:
		return 1
	case 1:
		return -1
	default:
		return 0
	}
}

func (r *EventsRepo) GetEvent(ctx context.Context, eventID string) (models.Event, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, pubkey, created_at, kind, tags, content, sig
		FROM events WHERE id = $1
	`, eventID)

	var event models.Event
	var tagsJSON []byte
	if err := row.Scan(&event.ID, &event.PubKey, &event.CreatedAt, &event.Kind, &tagsJSON, &event.Content, &event.Sig); err != nil {
		return models.Event{}, err
	}
	if err := json.Unmarshal(tagsJSON, &event.Tags); err != nil {
		return models.Event{}, fmt.Errorf("unmarshal tags: %w", err)
	}
	return event, nil
}

func (r *EventsRepo) MarkDeleted(ctx context.Context, deleted models.DeletedEvent) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO deleted_events (event_id, deleted_at, deleted_by, reason)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id) DO UPDATE
		SET deleted_at = EXCLUDED.deleted_at,
			deleted_by = EXCLUDED.deleted_by,
			reason = EXCLUDED.reason
	`, deleted.EventID, deleted.DeletedAt, deleted.DeletedBy, deleted.Reason)
	if err != nil {
		return fmt.Errorf("upsert deleted event: %w", err)
	}
	return nil
}

func (r *EventsRepo) QueryEvents(ctx context.Context, filter EventFilter) ([]models.Event, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	var builder strings.Builder
	args := make([]any, 0, 8)
	argIdx := 1

	builder.WriteString(`
		SELECT e.id, e.pubkey, e.created_at, e.kind, e.tags, e.content, e.sig
		FROM events e
	`)

	if !filter.IncludeDeleted {
		builder.WriteString("LEFT JOIN deleted_events d ON d.event_id = e.id\n")
	}

	builder.WriteString("WHERE 1=1\n")

	if !filter.IncludeDeleted {
		builder.WriteString("AND d.event_id IS NULL\n")
	}

	if filter.Author != "" {
		builder.WriteString(fmt.Sprintf("AND e.pubkey = $%d\n", argIdx))
		args = append(args, filter.Author)
		argIdx++
	}

	if filter.Kind != nil {
		builder.WriteString(fmt.Sprintf("AND e.kind = $%d\n", argIdx))
		args = append(args, *filter.Kind)
		argIdx++
	}

	if filter.Since != nil {
		builder.WriteString(fmt.Sprintf("AND e.created_at >= $%d\n", argIdx))
		args = append(args, *filter.Since)
		argIdx++
	}

	if filter.Until != nil {
		if strings.TrimSpace(filter.UntilID) != "" {
			builder.WriteString(fmt.Sprintf("AND (e.created_at < $%d OR (e.created_at = $%d AND e.id > $%d))\n", argIdx, argIdx, argIdx+1))
			args = append(args, *filter.Until, filter.UntilID)
			argIdx += 2
		} else {
			builder.WriteString(fmt.Sprintf("AND e.created_at <= $%d\n", argIdx))
			args = append(args, *filter.Until)
			argIdx++
		}
	}

	if filter.Tag != "" {
		tagName, tagValue := parseTagFilter(filter.Tag)
		if tagName != "" {
			builder.WriteString(fmt.Sprintf(`AND EXISTS (
				SELECT 1 FROM event_tags et
				WHERE et.event_id = e.id AND et.tag_name = $%d AND et.tag_value = $%d
			)
`, argIdx, argIdx+1))
			args = append(args, tagName, tagValue)
			argIdx += 2
		} else {
			builder.WriteString(fmt.Sprintf(`AND EXISTS (
				SELECT 1 FROM event_tags et
				WHERE et.event_id = e.id AND et.tag_value = $%d
			)
`, argIdx))
			args = append(args, tagValue)
			argIdx++
		}
	}

	builder.WriteString("ORDER BY e.created_at DESC, e.id ASC\n")
	builder.WriteString(fmt.Sprintf("LIMIT $%d", argIdx))
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, builder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	events := make([]models.Event, 0)
	for rows.Next() {
		var event models.Event
		var tagsJSON []byte
		if err := rows.Scan(&event.ID, &event.PubKey, &event.CreatedAt, &event.Kind, &tagsJSON, &event.Content, &event.Sig); err != nil {
			return nil, fmt.Errorf("scan event row: %w", err)
		}
		if err := json.Unmarshal(tagsJSON, &event.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal event tags: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate event rows: %w", err)
	}

	return events, nil
}

func parseTagFilter(raw string) (string, string) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", strings.TrimSpace(raw)
}

func (r *EventsRepo) insertEventTx(ctx context.Context, tx pgx.Tx, event models.Event) error {
	encodedTags, err := json.Marshal(event.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO events (id, pubkey, created_at, kind, tags, content, sig)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.ID, event.PubKey, event.CreatedAt, event.Kind, encodedTags, event.Content, event.Sig)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	normalizedTags := r.tagsRepo.Normalize(event.ID, event.Tags)
	for _, tag := range normalizedTags {
		tagArray, err := json.Marshal(tag.TagArray)
		if err != nil {
			return fmt.Errorf("marshal normalized tag: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_tags (event_id, tag_index, tag_name, tag_value, tag_array)
			VALUES ($1, $2, $3, $4, $5)
		`, tag.EventID, tag.TagIndex, tag.TagName, tag.TagValue, tagArray); err != nil {
			return fmt.Errorf("insert normalized tag: %w", err)
		}
	}

	return nil
}
