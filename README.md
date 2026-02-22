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

1.  **Initial Context:** The LLM is given a list of skill categories.
2.  **Discovery:** When the user asks for a task (e.g., "convert this file"), the LLM can list skills in a relevant category or search.
3.  **Refinement:** The LLM retrieves the specific tool definition and parameters only for the relevant skill.
4.  **Execution:** The LLM invokes the tool.

This "pull" model scales to thousands of tools without impacting the initial context size.

## Features

-   **Minimalist Design:** Relies on standard `database/sql` interfaces.
-   **Schema-First:** Provides the exact SQL schema description for LLM prompts.
-   **Programmatic Registration:** Easy `Register` method to add or update skills programmatically with "Upsert" logic and auto-provisioning of categories.
-   **Transactional:** Ensures skill and parameter updates are atomic.
-   **Search & Discovery:** Built-in methods for listing categories, listing skills by category, and SQL-based search.

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

Use `Register` to add or update skills. It handles upserts automatically and creates categories if they don't exist.

```go
ctx := context.Background()

mySkill := skill.Skill{
    Name:        "weather_check",
    Description: "Checks the weather for a given location",
    Category:    "Weather", // Category will be created if it doesn't exist
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

### LLM Strategy

To effectively use this library with an LLM, follow this discovery flow:

1.  **Orient:** Call `ListCategories` to get an overview of available domains.
2.  **Drill Down:** Call `ListSkillsByCategory` to see specific tools within a relevant domain.
3.  **Search (Optional):** Use `SearchSkills` if the category isn't obvious.
4.  **Inspect:** Call `GetSkillDetail` to get the full parameter schema for a chosen skill.

#### Example Flow

```go
// 1. List Categories
categories, _ := store.ListCategories(ctx)
// Present categories to LLM...

// 2. List Skills in a Category (e.g., "Data")
skills, _ := store.ListSkillsByCategory(ctx, "Data")
// Present skill names/descriptions to LLM...

// 3. Get Details for execution
details, err := store.GetSkillDetail(ctx, "convert_format")
```

### SQL Integration

You can also inject the schema description into your system prompt for raw SQL capabilities:

```go
systemPrompt := "You have access to a SQL database of tools. Schema:\n" + store.GetSchemaDescription()
```

When the LLM generates a SQL query, execute it safely:

```go
// Example: LLM wants to find image tools
query := "SELECT * FROM skills WHERE description LIKE '%image%'"
// Execute query against db...
```
