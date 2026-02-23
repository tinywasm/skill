package skill

// LLMInstruction is the prompt template for LLMs.
const LLMInstruction = "Available: %s. Query catalog table for args schema. Example: SELECT args FROM catalog WHERE name='read';"

// Skill represents a tool available to the LLM.
type Skill struct {
	Name       string      `json:"name"`       // Matches 'name' in catalog
	Category   string      `json:"category"`   // Matches 'cat' in catalog (mapped)
	Info       string      `json:"info"`       // Matches 'info' in catalog
	Parameters []Parameter `json:"parameters"` // Matches 'args' in catalog (mapped)
}

// Parameter defines an argument for a Skill.
// JSON tags match the compact format used in the 'catalog' view to save tokens.
type Parameter struct {
	Name     string `json:"n"` // Name
	Type     string `json:"t"` // Type
	Info     string `json:"d"` // Description/Info
	Required bool   `json:"r"` // Required
}
