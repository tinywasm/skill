package skill

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides access to the skill database.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store with the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// SearchSkills searches for skills by name or description.
func (s *Store) SearchSkills(ctx context.Context, query string) ([]Skill, error) {
	q := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, category_id, name, description
		FROM skills
		WHERE name LIKE ? OR description LIKE ?
	`, q, q)
	if err != nil {
		return nil, fmt.Errorf("search skills: %w", err)
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var skill Skill
		if err := rows.Scan(&skill.ID, &skill.CategoryID, &skill.Name, &skill.Description); err != nil {
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
func (s *Store) GetSkillDetail(ctx context.Context, name string) (*Skill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			s.id, s.category_id, s.name, s.description,
			p.id, p.skill_id, p.name, p.type, p.description, p.is_required
		FROM skills s
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

		// We scan skill fields every time, but they should be the same.
		// Alternatively, we could scan them once if we knew we were on the first row.
		// But usually we just overwrite.
		if err := rows.Scan(
			&skill.ID, &skill.CategoryID, &skill.Name, &skill.Description,
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

// GetSchemaDescription returns a brief string describing the SQL schema so the LLM knows how to query the DB directly.
func (s *Store) GetSchemaDescription() string {
	return `
CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT
);

CREATE TABLE skills (
    id INTEGER PRIMARY KEY,
    category_id INTEGER REFERENCES categories(id),
    name TEXT NOT NULL,
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
