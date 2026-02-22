package skill

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides access to the skill database.
// It manages the storage and retrieval of skills, categories, and parameters.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store with the given database connection.
// It expects the database to be initialized with the schema provided by GetSchemaDescription.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ListCategories lists all available categories.
func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, description FROM categories ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		var description sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &description); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		c.Description = description.String
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return categories, nil
}

// ListSkillsByCategory lists all skills under a specific category.
func (s *Store) ListSkillsByCategory(ctx context.Context, categoryName string) ([]Skill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, c.name, s.name, s.description
		FROM skills s
		JOIN categories c ON s.category_id = c.id
		WHERE c.name = ?
		ORDER BY s.name
	`, categoryName)
	if err != nil {
		return nil, fmt.Errorf("list skills by category: %w", err)
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var skill Skill
		if err := rows.Scan(&skill.ID, &skill.Category, &skill.Name, &skill.Description); err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		skills = append(skills, skill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skills: %w", err)
	}
	return skills, nil
}

// SearchSkills searches for skills by name or description.
// It performs a case-insensitive search using SQL LIKE operator.
func (s *Store) SearchSkills(ctx context.Context, query string) ([]Skill, error) {
	q := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, c.name, s.name, s.description
		FROM skills s
		JOIN categories c ON s.category_id = c.id
		WHERE s.name LIKE ? OR s.description LIKE ?
	`, q, q)
	if err != nil {
		return nil, fmt.Errorf("search skills: %w", err)
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var skill Skill
		if err := rows.Scan(&skill.ID, &skill.Category, &skill.Name, &skill.Description); err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		skills = append(skills, skill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skills: %w", err)
	}
	return skills, nil
}

// GetSkillDetail retrieves a skill with all its parameters joined.
// It returns an error if the skill is not found.
func (s *Store) GetSkillDetail(ctx context.Context, name string) (*Skill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			s.id, c.name, s.name, s.description,
			p.id, p.skill_id, p.name, p.type, p.description, p.is_required
		FROM skills s
		JOIN categories c ON s.category_id = c.id
		LEFT JOIN parameters p ON s.id = p.skill_id
		WHERE s.name = ?
	`, name)
	if err != nil {
		return nil, fmt.Errorf("get skill detail: %w", err)
	}
	defer rows.Close()

	var skill *Skill
	for rows.Next() {
		if skill == nil {
			skill = &Skill{}
		}

		var (
			pID          sql.NullInt64
			pSkillID     sql.NullInt64
			pName        sql.NullString
			pType        sql.NullString
			pDescription sql.NullString
			pIsRequired  sql.NullBool
		)

		if err := rows.Scan(
			&skill.ID, &skill.Category, &skill.Name, &skill.Description,
			&pID, &pSkillID, &pName, &pType, &pDescription, &pIsRequired,
		); err != nil {
			return nil, fmt.Errorf("scan skill detail: %w", err)
		}

		if pID.Valid {
			skill.Parameters = append(skill.Parameters, Parameter{
				ID:          pID.Int64,
				SkillID:     pSkillID.Int64,
				Name:        pName.String,
				Type:        pType.String,
				Description: pDescription.String,
				IsRequired:  pIsRequired.Bool,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skill detail: %w", err)
	}

	if skill == nil {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	return skill, nil
}

// Register adds a new skill or updates an existing one by name.
// It ensures skill names are unique. If a skill with the same name exists,
// its description, category, and parameters are updated.
// This operation is transactional: parameters are replaced atomically with the skill update.
// The category is auto-provisioned if it does not exist.
func (s *Store) Register(ctx context.Context, skill Skill) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Upsert Category
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO categories (name, description) VALUES (?, '')
		ON CONFLICT(name) DO NOTHING
	`, skill.Category); err != nil {
		return fmt.Errorf("upsert category: %w", err)
	}

	var categoryID int64
	if err := tx.QueryRowContext(ctx, "SELECT id FROM categories WHERE name = ?", skill.Category).Scan(&categoryID); err != nil {
		return fmt.Errorf("get category id: %w", err)
	}

	// 2. Upsert skill using SQLite ON CONFLICT clause.
	query := `
		INSERT INTO skills (category_id, name, description)
		VALUES (?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			category_id = excluded.category_id,
			description = excluded.description
		RETURNING id
	`
	var skillID int64
	err = tx.QueryRowContext(ctx, query, categoryID, skill.Name, skill.Description).Scan(&skillID)
	if err != nil {
		return fmt.Errorf("upsert skill: %w", err)
	}

	// 3. Replace parameters: delete existing and insert new ones.
	_, err = tx.ExecContext(ctx, "DELETE FROM parameters WHERE skill_id = ?", skillID)
	if err != nil {
		return fmt.Errorf("delete parameters: %w", err)
	}

	if len(skill.Parameters) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO parameters (skill_id, name, type, description, is_required)
			VALUES (?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare parameter insert: %w", err)
		}
		defer stmt.Close()

		for _, p := range skill.Parameters {
			_, err := stmt.ExecContext(ctx, skillID, p.Name, p.Type, p.Description, p.IsRequired)
			if err != nil {
				return fmt.Errorf("insert parameter %s: %w", p.Name, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetSchemaDescription returns the SQL DDL commands to create the database schema.
// This includes tables for categories, skills, and parameters.
// The 'skills' table enforces a UNIQUE constraint on the 'name' column.
// The 'categories' table enforces a UNIQUE constraint on the 'name' column.
func (s *Store) GetSchemaDescription() string {
	return `
CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT
);

CREATE TABLE skills (
    id INTEGER PRIMARY KEY,
    category_id INTEGER REFERENCES categories(id),
    name TEXT NOT NULL UNIQUE,
    description TEXT
);

CREATE TABLE parameters (
    id INTEGER PRIMARY KEY,
    skill_id INTEGER REFERENCES skills(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    description TEXT,
    is_required BOOLEAN DEFAULT FALSE
);
`
}
