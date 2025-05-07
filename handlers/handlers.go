gopackage handlers

import (
	"github.com/yourusername/stripe-api/config"
)

// Handlers holds all HTTP handlers and their dependencies
type Handlers struct {
	config *config.Config
}

// NewHandlers creates a new Handlers instance
func NewHandlers(cfg *config.Config) *Handlers {
	return &Handlers{
		config: cfg,
	}
}