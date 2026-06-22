package tools

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) {
	r.tools[tool.Spec().Name] = tool
}

func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *Registry) Specs() []map[string]any {
	var specs []map[string]any

	for _, tool := range r.tools {
		spec := tool.Spec()
		specs = append(specs, map[string]any{
			"name":        spec.Name,
			"description": spec.Description,
			"schema":      spec.Parameters,
		})
	}
	return specs
}
