# Go Skill Discovery Library

A high-performance, token-efficient discovery system for LLMs.

## Overview

This library provides a minimalist, SQL-based mechanism for LLMs to discover and invoke tools ("skills"). It is designed to minimize token usage in the initial system prompt by providing a compact index and allowing the LLM to query for details on demand.

## Features

-   **Token-Efficient Index:** `GetIndex()` returns a highly compact string (e.g., `files(read,write), net(ping)`) for the initial context.
-   **SQL-Based Discovery:** LLMs can query the `catalog` view to retrieve detailed parameter schemas only when needed.
-   **Idempotent Registration:** `Register()` handles upserts and auto-provisions categories.
-   **Minimalist Schema:** Uses `cats`, `skills`, `params` tables and a `catalog` view.

## Usage

### Installation

```bash
go get github.com/tinywasm/skill
```

### Initialization

Initialize the store with any `*sql.DB` connection (e.g., `modernc.org/sqlite`).

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/tinywasm/skill"
	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", ":memory:") // or file path
	store := skill.NewStore(db)

	// Initialize schema
	if _, err := db.Exec(store.GetSchemaDescription()); err != nil {
		log.Fatal(err)
	}
}
```

### Registering Skills

```go
ctx := context.Background()

mySkill := skill.Skill{
    Category: "weather",
    Name:     "check",
    Info:     "Get current weather",
    Parameters: []skill.Parameter{
        {Name: "city", Type: "string", Info: "City name", Required: true},
    },
}

if err := store.Register(ctx, mySkill); err != nil {
    log.Fatal(err)
}
```

### LLM Integration

#### 1. Initial Context

Inject the compact index and instructions into your system prompt.

```go
index, _ := store.GetIndex(ctx)
// index example: "weather(check), files(read,write)"

systemPrompt := fmt.Sprintf(skill.LLMInstruction, index)
```

The `skill.LLMInstruction` constant provides the standard prompt:

> "Available: [INDEX]. Query catalog table for args schema. Example: SELECT args FROM catalog WHERE name='read';"

#### 2. Skill Discovery

When the LLM needs to use a tool, it can query the `catalog` view or you can use `Search()` programmatically.

```go
// Programmatic search
skills, _ := store.Search(ctx, "weather")
```

#### 3. Execution

The LLM can query for the arguments schema directly:

```sql
SELECT args FROM catalog WHERE name='check';
```

Returns JSON: `[{"n":"city","t":"string","r":true,"d":"City name"}]`

## Schema

-   **cats**: `id`, `name` (unique)
-   **skills**: `id`, `cat_id`, `name` (unique), `info`
-   **params**: `id`, `skill_id`, `name`, `type`, `info`, `req`
-   **catalog** (VIEW): `cat`, `name`, `info`, `args` (JSON array of parameters)
