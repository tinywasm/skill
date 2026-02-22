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
	if skills[0].Category != "Data" {
		t.Errorf("expected category 'Data', got '%s'", skills[0].Category)
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
	if skill.Category != "Data" {
		t.Errorf("expected category 'Data', got '%s'", skill.Category)
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

	// 1. Register a new skill with EXISTING category
	newSkill := Skill{
		Category:    "Test Cat",
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
	if savedSkill.Category != "Test Cat" {
		t.Errorf("expected category 'Test Cat', got '%s'", savedSkill.Category)
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
		Category:    "Test Cat",
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

	// 3. Register skill with NEW category (Auto-provisioning)
	autoSkill := Skill{
		Category:    "New Auto Category",
		Name:        "auto_skill",
		Description: "Skill with new category",
	}
	if err := store.Register(ctx, autoSkill); err != nil {
		t.Fatalf("Register auto-provisioning failed: %v", err)
	}

	savedSkill, err = store.GetSkillDetail(ctx, "auto_skill")
	if err != nil {
		t.Fatalf("GetSkillDetail failed: %v", err)
	}
	if savedSkill.Category != "New Auto Category" {
		t.Errorf("expected category 'New Auto Category', got '%s'", savedSkill.Category)
	}

	// Verify category was created
	var catCount int
	err = store.db.QueryRow("SELECT COUNT(*) FROM categories WHERE name = 'New Auto Category'").Scan(&catCount)
	if err != nil {
		t.Fatalf("failed to count categories: %v", err)
	}
	if catCount != 1 {
		t.Errorf("expected 1 category, got %d", catCount)
	}

	// 4. Verify uniqueness constraint directly
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM skills WHERE name = 'new_skill'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count skills: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 skill with name 'new_skill', got %d", count)
	}
}

func TestListCategories(t *testing.T) {
	store := setupTestDB(t)
	seedData(t, store.db) // Adds 'Data' (ID 1)

	// Add another category
	_, err := store.db.Exec(`INSERT INTO categories (name, description) VALUES ('Audio', 'Audio processing')`)
	if err != nil {
		t.Fatalf("failed to add category: %v", err)
	}

	ctx := context.Background()
	cats, err := store.ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}

	if len(cats) != 2 {
		t.Errorf("expected 2 categories, got %d", len(cats))
	}

	// Ordered by name: Audio, Data
	if cats[0].Name != "Audio" {
		t.Errorf("expected first category 'Audio', got '%s'", cats[0].Name)
	}
	if cats[1].Name != "Data" {
		t.Errorf("expected second category 'Data', got '%s'", cats[1].Name)
	}
}

func TestListSkillsByCategory(t *testing.T) {
	store := setupTestDB(t)
	seedData(t, store.db) // Adds 'Data' category and 'convert_format', 'list_files' skills

	ctx := context.Background()
	skills, err := store.ListSkillsByCategory(ctx, "Data")
	if err != nil {
		t.Fatalf("ListSkillsByCategory failed: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}
	// Ordered by name: convert_format, list_files
	if skills[0].Name != "convert_format" {
		t.Errorf("expected first skill 'convert_format', got '%s'", skills[0].Name)
	}
	if skills[1].Name != "list_files" {
		t.Errorf("expected second skill 'list_files', got '%s'", skills[1].Name)
	}

	// Test non-existent category
	skills, err = store.ListSkillsByCategory(ctx, "NonExistent")
	if err != nil {
		t.Fatalf("ListSkillsByCategory failed: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestRegisterIdempotency(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	skill1 := Skill{
		Category:    "Common",
		Name:        "skill1",
		Description: "desc",
	}
	skill2 := Skill{
		Category:    "Common", // Same category
		Name:        "skill2",
		Description: "desc",
	}

	if err := store.Register(ctx, skill1); err != nil {
		t.Fatalf("Register skill1 failed: %v", err)
	}
	if err := store.Register(ctx, skill2); err != nil {
		t.Fatalf("Register skill2 failed: %v", err)
	}

	// Verify only one category "Common" exists
	var count int
	err := store.db.QueryRow("SELECT COUNT(*) FROM categories WHERE name = 'Common'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count categories: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 category 'Common', got %d", count)
	}
}
