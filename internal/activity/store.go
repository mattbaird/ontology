package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/matthewbaird/ontology/internal/signals"
	"github.com/matthewbaird/ontology/internal/types"
)

// Store is the interface for reading and writing activity entries.
// ActivityEntry is NOT an Ent entity — it's stored in a separate
// partitioned Postgres table outside the ORM.
type Store interface {
	// WriteEntries writes one or more activity entries (one event → many entries).
	WriteEntries(ctx context.Context, entries []types.ActivityEntry) error

	// QueryByEntity returns activity entries for a specific entity.
	QueryByEntity(ctx context.Context, entityType, entityID string, opts QueryOptions) (entries []types.ActivityEntry, nextCursor string, totalCount int, err error)

	// Search performs full-text search across activity summaries.
	Search(ctx context.Context, query string, opts SearchOptions) (entries []types.ActivityEntry, totalCount int, err error)
}

// PostgresStore implements Store using a partitioned Postgres table.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgresStore.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// CreateTable creates the activity_entries table with monthly partitioning.
// This should be run during database migration, not at startup.
func (s *PostgresStore) CreateTable(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS activity_entries (
			event_id            TEXT NOT NULL,
			event_type          TEXT NOT NULL,
			occurred_at         TIMESTAMPTZ NOT NULL,
			indexed_entity_type TEXT NOT NULL,
			indexed_entity_id   TEXT NOT NULL,
			entity_role         TEXT NOT NULL,
			source_refs         JSONB NOT NULL DEFAULT '[]',
			summary             TEXT NOT NULL,
			category            TEXT NOT NULL,
			weight              TEXT NOT NULL,
			polarity            TEXT NOT NULL,
			payload             JSONB,
			PRIMARY KEY (indexed_entity_type, indexed_entity_id, occurred_at, event_id)
		) PARTITION BY RANGE (occurred_at);

		CREATE INDEX IF NOT EXISTS idx_activity_entity_time
			ON activity_entries (indexed_entity_type, indexed_entity_id, occurred_at DESC);

		CREATE INDEX IF NOT EXISTS idx_activity_entity_category_time
			ON activity_entries (indexed_entity_type, indexed_entity_id, category, occurred_at DESC);
	`)
	return err
}

// WriteEntries inserts activity entries into Postgres.
func (s *PostgresStore) WriteEntries(ctx context.Context, entries []types.ActivityEntry) error {
	if len(entries) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString(`INSERT INTO activity_entries (
		event_id, event_type, occurred_at, indexed_entity_type, indexed_entity_id,
		entity_role, source_refs, summary, category, weight, polarity, payload
	) VALUES `)

	args := make([]interface{}, 0, len(entries)*12)
	for i, e := range entries {
		if i > 0 {
			b.WriteString(", ")
		}
		base := i * 12
		b.WriteString(fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6,
			base+7, base+8, base+9, base+10, base+11, base+12,
		))

		refsJSON, _ := json.Marshal(e.SourceRefs)
		args = append(args,
			e.EventID, e.EventType, e.OccurredAt, e.IndexedEntityType, e.IndexedEntityID,
			e.EntityRole, refsJSON, e.Summary, e.Category, e.Weight, e.Polarity, e.Payload,
		)
	}

	b.WriteString(" ON CONFLICT DO NOTHING")
	_, err := s.db.ExecContext(ctx, b.String(), args...)
	return err
}

// QueryByEntity returns activity entries for a specific entity with filtering and pagination.
func (s *PostgresStore) QueryByEntity(ctx context.Context, entityType, entityID string, opts QueryOptions) ([]types.ActivityEntry, string, int, error) {
	if opts.Limit <= 0 || opts.Limit > 500 {
		opts.Limit = 100
	}

	var conditions []string
	var args []interface{}
	argN := 1

	conditions = append(conditions, fmt.Sprintf("indexed_entity_type = $%d", argN))
	args = append(args, entityType)
	argN++

	conditions = append(conditions, fmt.Sprintf("indexed_entity_id = $%d", argN))
	args = append(args, entityID)
	argN++

	if opts.Since != nil {
		conditions = append(conditions, fmt.Sprintf("occurred_at >= $%d", argN))
		args = append(args, *opts.Since)
		argN++
	}
	if opts.Until != nil {
		conditions = append(conditions, fmt.Sprintf("occurred_at <= $%d", argN))
		args = append(args, *opts.Until)
		argN++
	}
	if len(opts.Categories) > 0 {
		placeholders := make([]string, len(opts.Categories))
		for i, cat := range opts.Categories {
			placeholders[i] = fmt.Sprintf("$%d", argN)
			args = append(args, cat)
			argN++
		}
		conditions = append(conditions, fmt.Sprintf("category IN (%s)", strings.Join(placeholders, ", ")))
	}
	if opts.MinWeight != "" && opts.MinWeight != "info" {
		maxSeverity := signals.WeightSeverity(opts.MinWeight)
		weightValues := make([]string, 0)
		for w, s := range signals.WeightOrder {
			if s <= maxSeverity {
				weightValues = append(weightValues, w)
			}
		}
		if len(weightValues) > 0 {
			placeholders := make([]string, len(weightValues))
			for i, wv := range weightValues {
				placeholders[i] = fmt.Sprintf("$%d", argN)
				args = append(args, wv)
				argN++
			}
			conditions = append(conditions, fmt.Sprintf("weight IN (%s)", strings.Join(placeholders, ", ")))
		}
	}
	if opts.Cursor != "" {
		// Cursor is the occurred_at timestamp of the last result.
		cursorTime, err := time.Parse(time.RFC3339Nano, opts.Cursor)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("occurred_at < $%d", argN))
			args = append(args, cursorTime)
			argN++
		}
	}

	where := strings.Join(conditions, " AND ")
	query := fmt.Sprintf(
		`SELECT event_id, event_type, occurred_at, indexed_entity_type, indexed_entity_id,
			entity_role, source_refs, summary, category, weight, polarity, payload
		FROM activity_entries
		WHERE %s
		ORDER BY occurred_at DESC
		LIMIT $%d`, where, argN)
	args = append(args, opts.Limit+1) // fetch one extra for cursor

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", 0, fmt.Errorf("querying activity entries: %w", err)
	}
	defer rows.Close()

	var entries []types.ActivityEntry
	for rows.Next() {
		var e types.ActivityEntry
		var refsJSON, payloadJSON []byte
		err := rows.Scan(
			&e.EventID, &e.EventType, &e.OccurredAt, &e.IndexedEntityType, &e.IndexedEntityID,
			&e.EntityRole, &refsJSON, &e.Summary, &e.Category, &e.Weight, &e.Polarity, &payloadJSON,
		)
		if err != nil {
			return nil, "", 0, fmt.Errorf("scanning activity entry: %w", err)
		}
		if len(refsJSON) > 0 {
			_ = json.Unmarshal(refsJSON, &e.SourceRefs)
		}
		e.Payload = payloadJSON
		entries = append(entries, e)
	}

	var nextCursor string
	if len(entries) > opts.Limit {
		entries = entries[:opts.Limit]
		nextCursor = entries[len(entries)-1].OccurredAt.Format(time.RFC3339Nano)
	}

	// Get total count (separate query for accuracy).
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM activity_entries WHERE %s", where)
	countArgs := args[:len(args)-1] // remove the LIMIT arg
	var totalCount int
	_ = s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount)

	return entries, nextCursor, totalCount, nil
}

// Search performs full-text search across activity summaries.
func (s *PostgresStore) Search(ctx context.Context, query string, opts SearchOptions) ([]types.ActivityEntry, int, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	var conditions []string
	var args []interface{}
	argN := 1

	// Full-text search on summary.
	conditions = append(conditions, fmt.Sprintf("summary ILIKE '%%' || $%d || '%%'", argN))
	args = append(args, query)
	argN++

	if opts.EntityType != "" {
		conditions = append(conditions, fmt.Sprintf("indexed_entity_type = $%d", argN))
		args = append(args, opts.EntityType)
		argN++
	}
	if opts.Since != nil {
		conditions = append(conditions, fmt.Sprintf("occurred_at >= $%d", argN))
		args = append(args, *opts.Since)
		argN++
	}
	if len(opts.Categories) > 0 {
		placeholders := make([]string, len(opts.Categories))
		for i, cat := range opts.Categories {
			placeholders[i] = fmt.Sprintf("$%d", argN)
			args = append(args, cat)
			argN++
		}
		conditions = append(conditions, fmt.Sprintf("category IN (%s)", strings.Join(placeholders, ", ")))
	}

	where := strings.Join(conditions, " AND ")
	sqlQuery := fmt.Sprintf(
		`SELECT event_id, event_type, occurred_at, indexed_entity_type, indexed_entity_id,
			entity_role, source_refs, summary, category, weight, polarity, payload
		FROM activity_entries
		WHERE %s
		ORDER BY occurred_at DESC
		LIMIT $%d`, where, argN)
	args = append(args, opts.Limit)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("searching activity entries: %w", err)
	}
	defer rows.Close()

	var entries []types.ActivityEntry
	for rows.Next() {
		var e types.ActivityEntry
		var refsJSON, payloadJSON []byte
		err := rows.Scan(
			&e.EventID, &e.EventType, &e.OccurredAt, &e.IndexedEntityType, &e.IndexedEntityID,
			&e.EntityRole, &refsJSON, &e.Summary, &e.Category, &e.Weight, &e.Polarity, &payloadJSON,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning activity entry: %w", err)
		}
		if len(refsJSON) > 0 {
			_ = json.Unmarshal(refsJSON, &e.SourceRefs)
		}
		e.Payload = payloadJSON
		entries = append(entries, e)
	}

	// Count total.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM activity_entries WHERE %s", where)
	countArgs := args[:len(args)-1]
	var totalCount int
	_ = s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount)

	return entries, totalCount, nil
}
