package chatui

import (
	"os"

	"goc/ccb-engine/gemma"
)

// ConfigFromEnv starts from [gemma.DefaultConfig] and overrides from env when set.
// GEMMA_PROJECT_ID, GEMMA_LOCATION, GEMMA_ENDPOINT_ID, GEMMA_MODEL_NAME, GEMMA_DEDICATED_DOMAIN.
func ConfigFromEnv() gemma.Config {
	c := gemma.DefaultConfig()
	if v := os.Getenv("GEMMA_PROJECT_ID"); v != "" {
		c.ProjectID = v
	}
	if v := os.Getenv("GEMMA_LOCATION"); v != "" {
		c.Location = v
	}
	if v := os.Getenv("GEMMA_ENDPOINT_ID"); v != "" {
		c.EndpointID = v
	}
	if v := os.Getenv("GEMMA_MODEL_NAME"); v != "" {
		c.ModelName = v
	}
	if v := os.Getenv("GEMMA_DEDICATED_DOMAIN"); v != "" {
		c.DedicatedDomain = v
	}
	return c
}
