package builtin

import (
	"github.com/cheetahbyte/apex/internal/tools"
	"github.com/cheetahbyte/apex/internal/tools/webfetch"
)

// NewRegistry returns Apex's built-in tool registry.
func NewRegistry() *tools.Registry {
	r := tools.NewRegistry()
	r.Register(webfetch.WebfetchTool{})
	return r
}
