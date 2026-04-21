package config

// ModelEntry defines which provider owns a model and what to failover to.
type ModelEntry struct {
	Provider string
	Failover []string // provider names to try in order on 429/5xx
}

// ModelCatalog maps model IDs to their provider and failover chain.
var ModelCatalog = map[string]ModelEntry{
	// OpenAI
	"gpt-4o":                {Provider: "openai", Failover: []string{"groq"}},
	"gpt-4o-mini":           {Provider: "openai", Failover: []string{"groq"}},
	"gpt-4-turbo":           {Provider: "openai", Failover: []string{}},

	// Anthropic
	"claude-3-5-sonnet-20241022": {Provider: "anthropic", Failover: []string{}},
	"claude-3-5-haiku-20241022":  {Provider: "anthropic", Failover: []string{}},
	"claude-3-opus-20240229":     {Provider: "anthropic", Failover: []string{}},

	// Google
	"gemini-1.5-pro":        {Provider: "google", Failover: []string{}},
	"gemini-1.5-flash":      {Provider: "google", Failover: []string{}},
	"gemini-2.0-flash":      {Provider: "google", Failover: []string{}},
	"gemini-2.0-flash-lite": {Provider: "google", Failover: []string{}},

	// Groq (fast inference)
	"llama-3.3-70b-versatile": {Provider: "groq", Failover: []string{"openai"}},
	"mixtral-8x7b-32768":      {Provider: "groq", Failover: []string{}},
}

func LookupModel(modelID string) (ModelEntry, bool) {
	entry, ok := ModelCatalog[modelID]
	return entry, ok
}
