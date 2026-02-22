# Go Skill Discovery Library

A minimalist Go library for SQL-based MCP (Model Context Protocol) skill discovery and LLM context optimization.

## Overview

This library provides a robust mechanism for storing, discovering, and managing "skills" (tools or capabilities) that can be invoked by Large Language Models (LLMs). Instead of flooding the LLM's context window with every available tool definition, this library enables an **SQL-based Skill Discovery** pattern.

### The Problem: Context Saturation

In traditional MCP implementations, all available tools are often described in the system prompt. As the number of tools grows:
1.  **Context Window Limits:** You run out of space for actual conversation history.
2.  **Cost:** Token usage increases with every request.
3.  **Confusion:** The LLM may hallucinate or get confused by irrelevant tools.

### The Solution: SQL-Based Skill Discovery

This library empowers the LLM to **query** for skills when needed.

1.  **Initial Context:** The LLM is given a brief description of the SQL schema (via `GetSchemaDescription`) and told it can query the `skills` table.
2.  **Discovery:** When the user asks for a task (e.g., "convert this file"), the LLM writes a SQL query to find relevant skills (e.g., `SELECT * FROM skills WHERE name LIKE '%convert%'`).
3.  **Refinement:** The LLM retrieves the specific tool definition and parameters only for the relevant skill.
4.  **Execution:** The LLM invokes the tool.

This "pull" model scales to thousands of tools without impacting the initial context size.

## Features

-   **Minimalist Design:** Relies on standard `database/sql` interfaces.
-   **Schema-First:** Provides the exact SQL schema description for LLM prompts.
-   **Programmatic Registration:** Easy `Register` method to add or update skills programmatically with "Upsert" logic.
-   **Transactional:** Ensures skill and parameter updates are atomic.
-   **Search:** Built-in SQL-based search for manual or heuristic discovery.

## Usage

### Installation

```bash
go get github.com/tinywasm/skill
```

### Initialization

Initialize the store with any `*sql.DB` connection (e.g., SQLite).

```go
package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/tinywasm/skill"
	_ "modernc.org/sqlite" // or any other driver
)

func main() {
	db, _ := sql.Open("sqlite", "skills.db")
	store := skill.NewStore(db)

	// Initialize schema
	if _, err := db.Exec(store.GetSchemaDescription()); err != nil {
		log.Fatal(err)
	}
}
```

### Registering Skills

Use `Register` to add or update skills. It handles upserts automatically.

```go
ctx := context.Background()

mySkill := skill.Skill{
    Name:        "weather_check",
    Description: "Checks the weather for a given location",
    CategoryID:  1, // Assuming category 1 exists
    Parameters: []skill.Parameter{
        {
            Name:        "location",
            Type:        "string",
            Description: "City name or coordinates",
            IsRequired:  true,
        },
    },
}

if err := store.Register(ctx, mySkill); err != nil {
    log.Printf("Failed to register skill: %v", err)
}
```

### LLM Integration

Inject the schema description into your system prompt:

```go
systemPrompt := "You have access to a SQL database of tools. Schema:\n" + store.GetSchemaDescription()
```

When the LLM generates a SQL query, execute it safely:

```go
// Example: LLM wants to find image tools
query := "SELECT * FROM skills WHERE description LIKE '%image%'"
// Execute query against db...
```

Or use the helper `SearchSkills` for simple keyword searches:

```go
skills, _ := store.SearchSkills(ctx, "image")
```

### Retrieving Details

Once a skill is selected, get full details including parameters:

```go
details, err := store.GetSkillDetail(ctx, "weather_check")
```
