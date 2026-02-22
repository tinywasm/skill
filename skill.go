package skill

// Category represents a grouping of skills.
type Category struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Skill represents a capability that can be performed.
type Skill struct {
	ID          int64       `json:"id"`
	CategoryID  int64       `json:"category_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []Parameter `json:"parameters,omitempty"`
}

// Parameter represents an argument for a skill.
type Parameter struct {
	ID          int64  `json:"id"`
	SkillID     int64  `json:"skill_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	IsRequired  bool   `json:"is_required"`
}
