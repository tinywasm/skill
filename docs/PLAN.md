# Implementation Plan - ORM and SQLite Migration

## Development Rules

- **Single Responsibility Principle (SRP):** Every file (CSS, Go, JS) must have a single, well-defined purpose.
- **Mandatory Dependency Injection (DI):** No global state. Inject `orm.DB` or `sqlite` interfaces.
- **Flat Hierarchy:** Go libraries must avoid subdirectories. Keep files in the root.
- **No Global State:** No direct system calls in logic.
- **Standard Library Only:** No external assertion libraries.
- **Testing Runner (`gotest`):** Use the globally installed `gotest` command.
- **Publishing (`gopush`):** Use `gopush` for commits and tags.
- **Frontend Go Compatibility:** Use `tinywasm/fmt` instead of `fmt`; `tinywasm/time` instead of `time`; and `tinywasm/json` instead of `encoding/json`.
- **Frontend Optimization:** Avoid using `map` declarations in WASM code.

## Context & References

- **ORM Repository:** [github.com/tinywasm/orm](https://github.com/tinywasm/orm) - Full documentation and examples.

### ORM Reference (tinywasm/orm)

#### Model Interface
```go
type Model interface {
    TableName() string
    Schema() []orm.Field   // Key: column name, includes type + constraints
    Values() []any
    Pointers() []any
}
```

#### Schema Constraints (`db` tags)
| Constant | db tag | Notes |
|---|---|---|
| `ConstraintPK` | `db:"pk"` | Auto-detected via `tinywasm/fmt.IDorPrimaryKey` |
| `ConstraintUnique` | `db:"unique"` | |
| `ConstraintNotNull` | `db:"not_null"` | |
| `ConstraintAutoIncrement` | `db:"autoincrement"` | Numeric fields only |
| FK reference | `db:"ref=table"` or `db:"ref=table:column"` | `Field.Ref` + `Field.RefColumn` |
| Ignore field | `db:"-"` | Silently excluded from `Schema()`, `Values()`, `Pointers()` |

#### Auto-Generated Code (`cmd/ormc`)
Run `ormc` from the project root to generate `model_orm.go`. It provides:
- `func (m *User) Schema() []orm.Field`
- `func (m *User) Values() []any`
- `func (m *User) Pointers() []any`
- `func (m *User) TableName() string`
- `UserMeta` struct with typed column name constants
- `ReadAllUser(qb *orm.QB) ([]*User, error)`

#### Schema DDL
```go
if err := db.CreateTable(&User{}); err != nil { ... }
if err := db.DropTable(&User{}); err != nil { ... }
```


## Strategy

1.  **Persistence Layer**: Migrate from manual `sql.DB` to `orm.DB`.
2.  **Schema Definition**: Replace the `GetSchemaDescription` string with auto-generated ORM models in `models.go`.
3.  **Discovery View**: Keep the `catalog` view for LLM token optimization, but manage its lifecycle via the ORM/SQLite adapter.
4.  **WASM Compatibility**: Refactor existing logic in `repository.go` and `skill.go` to use `tinywasm` replacements for standard libraries.

## Prerequisites

External agents must install the following tools before proceeding:

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
go install github.com/tinywasm/orm/cmd/ormc@latest
```

## Steps

### 1. Dependency Update
- Update `go.mod` to include:
    - `github.com/tinywasm/orm`
    - `github.com/tinywasm/sqlite`
    - `github.com/tinywasm/fmt`
- Remove `modernc.org/sqlite` from `go.mod`.

### 2. Define ORM Models
- Create `models.go` in the root:
    - Define `cat` struct with tags for `pk` and `unique`. Table name: `cats`.
    - Define `skillModel` struct with tags for `pk`, `ref=cats`, and `unique`. Table name: `skills`.
    - Define `paramModel` struct with tags for `pk` and `ref=skills`. Table name: `params`.
- Tag existing `Skill` and `Parameter` structs in `skill.go` if they can serve as models or keep them for the public API and map them.
- *Decision*: Map public `Skill`/`Parameter` to hidden `skillModel`/`paramModel` in `repository.go` to maintain the existing public API surface and special JSON tagging.

### 3. Generate ORM Code
- Add `//go:generate ormc` to `repository.go`.
- Run `ormc` to generate persistence methods.

### 4. Refactor Repository Logic (`repository.go`)
- Update `Store` struct: change `db *sql.DB` to `db *orm.DB`.
- Refactor `Register`:
    - Use `db.Tx` for atomicity.
    - Implement upsert logic using `db.Create` and `db.Update` via the generated models.
- Refactor `Search`:
    - Query the `catalog` view or fetch `skillModel` with relations.
- Refactor `GetIndex`:
    - Fetch all categories and skills, avoiding `map` (use slices/structs for WASM compatibility).
- Remove `GetSchemaDescription`.

### 5. Standard Library Migration
- Replace `fmt`, `strings`, `errors` with `tinywasm/fmt` counterparts.
- Replace `encoding/json` with `tinywasm/json`.
- Replace `time` if used.

### 6. Update Tests (`repository_test.go`)
- Change `setupTestDB` to use `sqlite.Open(":memory:")` returning `*orm.DB`.
- Replace manual schema creation with Calls to `db.CreateTable` for all models.
- Re-create the `catalog` view via `sqlite.ExecSQL`.
- Replace `modernc.org/sqlite` import with `github.com/tinywasm/sqlite`.

### 7. Documentation Sync
- Update `README.md` examples to show `orm.DB` and `tinywasm/sqlite` usage.

## Verification
- Run `gotest` to ensure all existing tests pass with the new database backend.
- Check code coverage to ensure it remains high (>90%).
