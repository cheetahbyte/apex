package providers

import (
	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/config"
)

type Protocol string

const (
	ProtocolOpenAICompatible Protocol = "openai-compatible"
)

type Provider struct {
	ID           string
	DisplayName  string
	Protocol     Protocol
	BaseURL      string
	DefaultModel string
	AuthKind     auth.AuthKind
	ToolMode     config.ToolMode
}
