package skill

import (
	"context"
	"strings"
	"fmt"

	"github.com/tinywasm/json"
	"github.com/tinywasm/orm"
)

// Store provides access to the skill database.
type Store struct {
	db *orm.DB
}

// NewStore creates a new Store with the given database connection.
func NewStore(db *orm.DB) *Store {
	return &Store{db: db}
}

// GetIndex returns a compact string representation of available skills.
// Format: cat1(skill1,skill2), cat2(skill3)
func (s *Store) GetIndex(ctx context.Context) (string, error) {
	type row struct {
		Cat   string `db:"c.name"`
		Skill string `db:"s.name"`
	}

	// Query all skills joined with categories.
	// Ordering by category and skill name ensures deterministic output.
	rows, err := s.db.RawExecutor().Query(`
		SELECT c.name, s.name
		FROM skills s
		JOIN cats c ON s.cat_id = c.id
		ORDER BY c.name, s.name
	`)
	if err != nil {
		return "", fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()

	// Group skills by category without map for WASM compatibility
	type catGroup struct {
		name   string
		skills []string
	}
	var groups []catGroup

	for rows.Next() {
		var r row
		if err := rows.Scan(&r.Cat, &r.Skill); err != nil {
			return "", fmt.Errorf("scan list skills: %w", err)
		}
		found := false
		for i := range groups {
			if groups[i].name == r.Cat {
				groups[i].skills = append(groups[i].skills, r.Skill)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, catGroup{name: r.Cat, skills: []string{r.Skill}})
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("list skills iter: %w", err)
	}

	// Build the string
	var sb strings.Builder
	for i, g := range groups {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(g.name)
		sb.WriteString("(")
		sb.WriteString(strings.Join(g.skills, ","))
		sb.WriteString(")")
	}

	return sb.String(), nil
}

// Search queries the catalog view for skills matching the query.
// It maps the JSON 'args' column to the Skill.Parameters field.
func (s *Store) Search(ctx context.Context, query string) ([]Skill, error) {
	q := "%" + query + "%"

	type row struct {
		Cat  string `db:"cat"`
		Name string `db:"name"`
		Info string `db:"info"`
		Args string `db:"args"`
	}

	rows, err := s.db.RawExecutor().Query(`
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
		var r row
		if err := rows.Scan(&r.Cat, &r.Name, &r.Info, &r.Args); err != nil {
			return nil, fmt.Errorf("scan search skills: %w", err)
		}
		var sk Skill
		sk.Category = r.Cat
		sk.Name = r.Name
		sk.Info = r.Info

		if len(r.Args) > 0 {
			if err := json.Decode([]byte(r.Args), &sk.Parameters); err != nil {
				return nil, fmt.Errorf("unmarshal args for skill %s: %w", sk.Name, err)
			}
		}
		skills = append(skills, sk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search skills iter: %w", err)
	}

	return skills, nil
}

// Register adds a new skill or updates an existing one.
// It is idempotent, auto-creates categories, upserts the skill, and replaces parameters.
func (s *Store) Register(ctx context.Context, skill Skill) error {
	return s.db.Tx(func(tx *orm.DB) error {
		// 1. Auto-create category (cats table)
		// We use tx.RawExecutor().Exec to handle the ON CONFLICT DO NOTHING
		if err := tx.RawExecutor().Exec(`
			INSERT INTO cats (name) VALUES (?)
			ON CONFLICT(name) DO NOTHING
		`, skill.Category); err != nil {
			return fmt.Errorf("upsert category: %w", err)
		}

		// Get the category ID
		cats, err := ReadAllcat(tx.Query(&cat{}).Where(catMeta.Name).Eq(skill.Category))
		if err != nil || len(cats) == 0 {
			return fmt.Errorf("get category id: %w", err)
		}
		catID := cats[0].ID

		// 2. Upsert skill (skills table)
		if err := tx.RawExecutor().Exec(`
			INSERT INTO skills (cat_id, name, info)
			VALUES (?, ?, ?)
			ON CONFLICT(name) DO UPDATE SET
				cat_id = excluded.cat_id,
				info = excluded.info
		`, catID, skill.Name, skill.Info); err != nil {
			return fmt.Errorf("upsert skill: %w", err)
		}

		// Get the skill ID
		skills, err := ReadAllskillModel(tx.Query(&skillModel{}).Where(skillModelMeta.Name).Eq(skill.Name))
		if err != nil || len(skills) == 0 {
			return fmt.Errorf("get skill id: %w", err)
		}
		skillID := skills[0].ID

		// 3. Replace parameters (params table)
		// Delete existing parameters for this skill
		if err := tx.RawExecutor().Exec("DELETE FROM params WHERE skill_id = ?", skillID); err != nil {
			return fmt.Errorf("delete params: %w", err)
		}

		// Insert new parameters
		for _, p := range skill.Parameters {
			if err := tx.RawExecutor().Exec(`
				INSERT INTO params (skill_id, name, type, info, req)
				VALUES (?, ?, ?, ?, ?)
			`, skillID, p.Name, p.Type, p.Info, p.Required); err != nil {
				return fmt.Errorf("insert param: %w", err)
			}
		}

		return nil
	})
}
