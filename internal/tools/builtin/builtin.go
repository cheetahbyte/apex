package builtin

import (
	"github.com/cheetahbyte/apex/internal/skills"
	"github.com/cheetahbyte/apex/internal/tools"
	"github.com/cheetahbyte/apex/internal/tools/files"
	skilltool "github.com/cheetahbyte/apex/internal/tools/skills"
	"github.com/cheetahbyte/apex/internal/tools/webfetch"
)

// NewRegistry returns Apex's built-in tool registry.
func NewRegistry() *tools.Registry {
	return NewRegistryWithSkills(nil)
}

func NewRegistryWithSkills(skillStore *skills.Store) *tools.Registry {
	r := tools.NewRegistry()
	r.Register(files.ReadFileTool{})
	r.Register(webfetch.WebfetchTool{})
	r.Register(files.TreeTool{})
	if skillStore != nil {
		r.Register(skilltool.New(skillStore))
	}
	return r
}
