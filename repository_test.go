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

func TestRegisterAndGetIndex(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Register skills
	skills := []Skill{
		{
			Category: "files",
			Name:     "read",
			Info:     "Read file content",
			Parameters: []Parameter{
				{Name: "path", Type: "string", Info: "File path", Required: true},
			},
		},
		{
			Category: "files",
			Name:     "write",
			Info:     "Write to file",
		},
		{
			Category: "net",
			Name:     "ping",
			Info:     "Ping host",
		},
	}

	for _, s := range skills {
		if err := store.Register(ctx, s); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	// Test GetIndex
	index, err := store.GetIndex(ctx)
	if err != nil {
		t.Fatalf("GetIndex failed: %v", err)
	}

	expected := "files(read,write), net(ping)"
	if index != expected {
		t.Errorf("GetIndex mismatch.\nGot:  %s\nWant: %s", index, expected)
	}
}

func TestRegisterIdempotency(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	skill := Skill{
		Category: "test",
		Name:     "dup",
		Info:     "Original",
		Parameters: []Parameter{
			{Name: "p1", Type: "int", Info: "Param 1", Required: true},
		},
	}

	// First registration
	if err := store.Register(ctx, skill); err != nil {
		t.Fatalf("First Register failed: %v", err)
	}

	// Update skill (change info and params)
	skill.Info = "Updated"
	skill.Parameters = []Parameter{
		{Name: "p2", Type: "string", Info: "Param 2", Required: false},
	}

	// Second registration
	if err := store.Register(ctx, skill); err != nil {
		t.Fatalf("Second Register failed: %v", err)
	}

	// Verify update
	res, err := store.Search(ctx, "dup")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(res))
	}

	s := res[0]
	if s.Info != "Updated" {
		t.Errorf("expected info 'Updated', got '%s'", s.Info)
	}
	if len(s.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(s.Parameters))
	}
	p := s.Parameters[0]
	if p.Name != "p2" {
		t.Errorf("expected param name 'p2', got '%s'", p.Name)
	}
	if p.Required != false {
		t.Errorf("expected param not required")
	}
}

func TestSearchAndCatalogView(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	s := Skill{
		Category: "search_test",
		Name:     "complex_tool",
		Info:     "A tool with params",
		Parameters: []Parameter{
			{Name: "arg1", Type: "string", Info: "Argument 1", Required: true},
			{Name: "arg2", Type: "int", Info: "Argument 2", Required: false},
		},
	}
	if err := store.Register(ctx, s); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Test Search
	skills, err := store.Search(ctx, "complex")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	found := skills[0]
	if found.Name != "complex_tool" {
		t.Errorf("expected name 'complex_tool', got '%s'", found.Name)
	}
	if found.Category != "search_test" {
		t.Errorf("expected category 'search_test', got '%s'", found.Category)
	}
	if len(found.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(found.Parameters))
	}

	// Verify parameter fields (JSON mapping)
	var pArg1, pArg2 Parameter
	for _, p := range found.Parameters {
		if p.Name == "arg1" {
			pArg1 = p
		} else if p.Name == "arg2" {
			pArg2 = p
		}
	}

	if pArg1.Name == "" || pArg2.Name == "" {
		t.Fatalf("missing parameters")
	}

	if pArg1.Required != true {
		t.Errorf("arg1 should be required")
	}
	if pArg2.Required != false {
		t.Errorf("arg2 should not be required")
	}
	if pArg1.Type != "string" {
		t.Errorf("arg1 type mismatch")
	}
	if pArg1.Info != "Argument 1" {
		t.Errorf("arg1 info mismatch")
	}
}

func TestSearchNoParams(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	s := Skill{
		Category: "noparam",
		Name:     "simple",
		Info:     "Simple skill",
	}
	if err := store.Register(ctx, s); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	skills, err := store.Search(ctx, "simple")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if len(skills[0].Parameters) != 0 {
		t.Errorf("expected 0 params, got %d", len(skills[0].Parameters))
	}
}
