package skill

// Category represents a logical grouping of skills.
// Categories help organize skills into manageable sets, useful for UI display
// or filtering.
type Category struct {
	// ID is the unique identifier for the category.
	ID int64 `json:"id"`
	// Name is the display name of the category.
	Name string `json:"name"`
	// Description provides more details about the category.
	Description string `json:"description"`
}

// Skill represents a specific capability or tool that can be executed.
// It contains metadata about the tool, including its name, description,
// and the parameters required to invoke it.
type Skill struct {
	// ID is the unique identifier for the skill.
	ID int64 `json:"id"`
	// CategoryID references the category this skill belongs to.
	CategoryID int64 `json:"category_id"`
	// Name is the unique name of the skill, used for invocation.
	Name string `json:"name"`
	// Description explains what the skill does.
	Description string `json:"description"`
	// Parameters is a list of arguments that the skill accepts.
	Parameters []Parameter `json:"parameters,omitempty"`
}

// Parameter represents an individual argument that must or can be provided
// when invoking a skill.
type Parameter struct {
	// ID is the unique identifier for the parameter.
	ID int64 `json:"id"`
	// SkillID references the skill this parameter belongs to.
	SkillID int64 `json:"skill_id"`
	// Name is the name of the parameter.
	Name string `json:"name"`
	// Type defines the data type of the parameter (e.g., "string", "integer").
	Type string `json:"type"`
	// Description explains the purpose of the parameter.
	Description string `json:"description"`
	// IsRequired indicates if the parameter must be provided.
	IsRequired bool `json:"is_required"`
}
