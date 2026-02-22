package skill

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *Store {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store := NewStore(db)

	schema := store.GetSchemaDescription()
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return store
}

func seedData(t *testing.T, db *sql.DB) {
	_, err := db.Exec(`
		INSERT INTO categories (id, name, description) VALUES (1, 'Data', 'Data processing');
		INSERT INTO skills (id, category_id, name, description) VALUES (1, 1, 'convert_format', 'Convert file format');
		INSERT INTO parameters (id, skill_id, name, type, description, is_required) VALUES (1, 1, 'source', 'string', 'Source format', 1);
		INSERT INTO parameters (id, skill_id, name, type, description, is_required) VALUES (2, 1, 'target', 'string', 'Target format', 1);

		INSERT INTO skills (id, category_id, name, description) VALUES (2, 1, 'list_files', 'List files in directory');
	`)
	if err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}
}

func TestSearchSkills(t *testing.T) {
	store := setupTestDB(t)
	seedData(t, store.db)

	ctx := context.Background()
	skills, err := store.SearchSkills(ctx, "convert")
	if err != nil {
		t.Fatalf("SearchSkills failed: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "convert_format" {
		t.Errorf("expected skill 'convert_format', got '%s'", skills[0].Name)
	}

	skills, err = store.SearchSkills(ctx, "files")
	if err != nil {
		t.Fatalf("SearchSkills failed: %v", err)
	}
	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "list_files" {
		t.Errorf("expected skill 'list_files', got '%s'", skills[0].Name)
	}
}

func TestGetSkillDetail(t *testing.T) {
	store := setupTestDB(t)
	seedData(t, store.db)

	ctx := context.Background()
	skill, err := store.GetSkillDetail(ctx, "convert_format")
	if err != nil {
		t.Fatalf("GetSkillDetail failed: %v", err)
	}

	if skill.Name != "convert_format" {
		t.Errorf("expected skill name 'convert_format', got '%s'", skill.Name)
	}
	if len(skill.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(skill.Parameters))
	}

	// Check parameter details
	foundSource := false
	foundTarget := false
	for _, p := range skill.Parameters {
		if p.Name == "source" {
			foundSource = true
			if !p.IsRequired {
				t.Errorf("expected source to be required")
			}
		} else if p.Name == "target" {
			foundTarget = true
			if !p.IsRequired {
				t.Errorf("expected target to be required")
			}
		}
	}
	if !foundSource {
		t.Errorf("expected parameter 'source' not found")
	}
	if !foundTarget {
		t.Errorf("expected parameter 'target' not found")
	}

	// Test skill with no parameters
	skill, err = store.GetSkillDetail(ctx, "list_files")
	if err != nil {
		t.Fatalf("GetSkillDetail failed: %v", err)
	}
	if skill.Name != "list_files" {
		t.Errorf("expected skill name 'list_files', got '%s'", skill.Name)
	}
	if len(skill.Parameters) != 0 {
		t.Errorf("expected 0 parameters, got %d", len(skill.Parameters))
	}
}

func TestGetSchemaDescription(t *testing.T) {
	store := setupTestDB(t)
	desc := store.GetSchemaDescription()
	if desc == "" {
		t.Errorf("expected schema description not to be empty")
	}
}

func TestRegister(t *testing.T) {
	store := setupTestDB(t)
	// Seed initial category
	_, err := store.db.Exec(`INSERT INTO categories (id, name, description) VALUES (1, 'Test Cat', 'Test Description')`)
	if err != nil {
		t.Fatalf("failed to seed category: %v", err)
	}

	ctx := context.Background()

	// 1. Register a new skill
	newSkill := Skill{
		CategoryID:  1,
		Name:        "new_skill",
		Description: "A new skill",
		Parameters: []Parameter{
			{Name: "param1", Type: "string", Description: "First parameter", IsRequired: true},
		},
	}

	if err := store.Register(ctx, newSkill); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify registration
	savedSkill, err := store.GetSkillDetail(ctx, "new_skill")
	if err != nil {
		t.Fatalf("GetSkillDetail failed: %v", err)
	}
	if savedSkill.Name != "new_skill" {
		t.Errorf("expected skill name 'new_skill', got '%s'", savedSkill.Name)
	}
	if savedSkill.Description != "A new skill" {
		t.Errorf("expected description 'A new skill', got '%s'", savedSkill.Description)
	}
	if len(savedSkill.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(savedSkill.Parameters))
	}
	if savedSkill.Parameters[0].Name != "param1" {
		t.Errorf("expected parameter 'param1', got '%s'", savedSkill.Parameters[0].Name)
	}

	// 2. Update existing skill (upsert)
	updatedSkill := Skill{
		CategoryID:  1,
		Name:        "new_skill", // Same name
		Description: "Updated description",
		Parameters: []Parameter{
			{Name: "param2", Type: "int", Description: "New parameter", IsRequired: false},
		},
	}

	if err := store.Register(ctx, updatedSkill); err != nil {
		t.Fatalf("Register update failed: %v", err)
	}

	// Verify update
	savedSkill, err = store.GetSkillDetail(ctx, "new_skill")
	if err != nil {
		t.Fatalf("GetSkillDetail failed: %v", err)
	}
	if savedSkill.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got '%s'", savedSkill.Description)
	}
	if len(savedSkill.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(savedSkill.Parameters))
	}
	if savedSkill.Parameters[0].Name != "param2" {
		t.Errorf("expected parameter 'param2', got '%s'", savedSkill.Parameters[0].Name)
	}

	// 3. Verify uniqueness constraint directly
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM skills WHERE name = 'new_skill'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count skills: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 skill with name 'new_skill', got %d", count)
	}
}
