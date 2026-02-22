This is a specialized prompt designed for **Jules** (or any expert Go agent). It follows your philosophy of minimal dependencies, uses the standard library's database/sql interface, and targets the specific repository and schema we've discussed.

# ---

**Prompt: Implement tinywasm/skill Discovery Library**

**Role:** Expert Go Developer (Jules).

**Project:** github.com/tinywasm/skill

**Context:** This library is a "Skill Discovery" layer for a custom MCP (Model Context Protocol) server. Instead of exposing all tool definitions in the LLM context, the LLM will use SQL to query this database to find the "Skills" it needs.

## **1\. Core Philosophy**

* **Minimalist Go:** Use only the standard library where possible.  
* **No Frameworks:** Avoid ORMs or heavy web frameworks.  
* **Dependency Injection:** The library must accept a \*sql.DB (or a compatible interface) to remain database-agnostic.  
* **Testing:** Use the pure-Go SQLite driver (e.g., modernc.org/sqlite) for realistic, file-based or in-memory integration tests.

## **2\. Domain Model & Schema**

Implement the following entities as Go structs with appropriate tags:

* **Category**: ID, Name, Description.  
* **Skill**: ID, CategoryID, Name, Description.  
* **Parameter**: ID, SkillID, Name, Type, Description, IsRequired.

**Schema Reference:**

1. categories (1) \-\> (N) skills  
2. skills (1) \-\> (N) parameters

## **3\. Implementation Requirements**

Follow the guidelines established in https://github.com/tinywasm/devflow/blob/main/docs/DEFAULT\_LLM\_SKILL.md:

1. **Repository/Store Pattern:** Create a Store or Repository struct in the skill package that encapsulates the SQL logic.  
2. **SQL Discovery Methods:**  
   * SearchSkills(ctx, query string) (\[\]Skill, error): Search skills by name or description.  
   * GetSkillDetail(ctx, name string) (\*Skill, error): Retrieve a skill with all its parameters joined.  
   * GetSchemaDescription() string: A helper that returns a brief string describing the SQL schema so the LLM knows how to query the DB directly.  
3. **Strict Typing:** Ensure Go types map correctly to the SQL schema (e.g., bool for is\_required).

## **4\. Testing Strategy**

* Provide a store\_test.go.  
* Initialize an in-memory SQLite database in TestMain or within specific test functions.  
* Verify that JOIN queries correctly populate the Parameters slice within the Skill struct.

## **5\. Instructions for Jules**

"Jules, please initialize the project at github.com/tinywasm/skill. Start by defining the types in skill.go, then implement the database logic in repository.go. Ensure the code is clean, documented, and follows the 'Effective Go' standards. Do not add external dependencies unless strictly necessary for the SQLite driver in tests."

### ---

**Next Step**

Would you like me to generate the **README.md** for this repository explaining to the LLM how it should use the SQL interface to find its skills?