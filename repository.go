package skill

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Store provides access to the skill database.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store with the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetIndex returns a compact string representation of available skills.
// Format: cat1(skill1,skill2), cat2(skill3)
func (s *Store) GetIndex(ctx context.Context) (string, error) {
	// Query all skills joined with categories.
	// Ordering by category and skill name ensures deterministic output.
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.name, s.name
		FROM skills s
		JOIN cats c ON s.cat_id = c.id
		ORDER BY c.name, s.name
	`)
	if err != nil {
		return "", fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()

	// Group skills by category
	catMap := make(map[string][]string)
	for rows.Next() {
		var cat, skill string
		if err := rows.Scan(&cat, &skill); err != nil {
			return "", fmt.Errorf("scan skill: %w", err)
		}
		catMap[cat] = append(catMap[cat], skill)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate skills: %w", err)
	}

	// Sort categories
	var cats []string
	for c := range catMap {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	// Build the string
	var sb strings.Builder
	for i, c := range cats {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(c)
		sb.WriteString("(")
		sb.WriteString(strings.Join(catMap[c], ","))
		sb.WriteString(")")
	}

	return sb.String(), nil
}

// Search queries the catalog view for skills matching the query.
// It maps the JSON 'args' column to the Skill.Parameters field.
func (s *Store) Search(ctx context.Context, query string) ([]Skill, error) {
	q := "%" + query + "%"
	// The view returns 'args' as a JSON string.
	rows, err := s.db.QueryContext(ctx, `
		SELECT cat, name, info, args
		FROM catalog
		WHERE name LIKE ? OR info LIKE ?
	`, q, q)
	if err != nil {
		return nil, fmt.Errorf("search skills: %w", err)
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var sk Skill
		var argsJSON []byte // Use []byte for direct unmarshal
		if err := rows.Scan(&sk.Category, &sk.Name, &sk.Info, &argsJSON); err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}

		if len(argsJSON) > 0 {
			if err := json.Unmarshal(argsJSON, &sk.Parameters); err != nil {
				return nil, fmt.Errorf("unmarshal args for skill %s: %w", sk.Name, err)
			}
		}
		skills = append(skills, sk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skills: %w", err)
	}

	return skills, nil
}

// Register adds a new skill or updates an existing one.
// It is idempotent, auto-creates categories, upserts the skill, and replaces parameters.
func (s *Store) Register(ctx context.Context, skill Skill) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Auto-create category (cats table)
	// We use INSERT OR IGNORE (or ON CONFLICT DO NOTHING) to handle duplicates.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO cats (name) VALUES (?)
		ON CONFLICT(name) DO NOTHING
	`, skill.Category); err != nil {
		return fmt.Errorf("upsert category: %w", err)
	}

	// Get the category ID
	var catID int64
	if err := tx.QueryRowContext(ctx, "SELECT id FROM cats WHERE name = ?", skill.Category).Scan(&catID); err != nil {
		return fmt.Errorf("get category id: %w", err)
	}

	// 2. Upsert skill (skills table)
	// We need the skill ID for parameter insertion.
	// Using ON CONFLICT(name) DO UPDATE to handle updates.
	var skillID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO skills (cat_id, name, info)
		VALUES (?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			cat_id = excluded.cat_id,
			info = excluded.info
		RETURNING id
	`, catID, skill.Name, skill.Info).Scan(&skillID)
	if err != nil {
		return fmt.Errorf("upsert skill: %w", err)
	}

	// 3. Replace parameters (params table)
	// Delete existing parameters for this skill
	if _, err := tx.ExecContext(ctx, "DELETE FROM params WHERE skill_id = ?", skillID); err != nil {
		return fmt.Errorf("delete params: %w", err)
	}

	// Insert new parameters
	if len(skill.Parameters) > 0 {
		// Prepare statement for bulk insert efficiency
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO params (skill_id, name, type, info, req)
			VALUES (?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare param insert: %w", err)
		}
		defer stmt.Close()

		for _, p := range skill.Parameters {
			if _, err := stmt.ExecContext(ctx, skillID, p.Name, p.Type, p.Info, p.Required); err != nil {
				return fmt.Errorf("insert param: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetSchemaDescription returns the SQL DDL commands to create the database schema.
func (s *Store) GetSchemaDescription() string {
	return `
CREATE TABLE cats (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE
);

CREATE TABLE skills (
    id INTEGER PRIMARY KEY,
    cat_id INTEGER REFERENCES cats(id),
    name TEXT UNIQUE,
    info TEXT
);

CREATE TABLE params (
    id INTEGER PRIMARY KEY,
    skill_id INTEGER REFERENCES skills(id),
    name TEXT,
    type TEXT,
    info TEXT,
    req BOOLEAN
);

CREATE VIEW catalog AS
SELECT
    c.name AS cat,
    s.name AS name,
    s.info AS info,
    (SELECT json_group_array(
        json_object('n', p.name, 't', p.type, 'r', CASE WHEN p.req THEN json('true') ELSE json('false') END, 'd', p.info)
    ) FROM params p WHERE p.skill_id = s.id) AS args
FROM skills s
JOIN cats c ON s.cat_id = c.id;
`
}
