package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Skill struct {
	Name        string
	Path        string
	Description string
	Trigger     string
}

type Store struct {
	byName  map[string]Skill
	ordered []Skill
}

func NewStore() *Store {
	return &Store{
		byName:  make(map[string]Skill),
		ordered: make([]Skill, 0),
	}
}

func LoadDefault() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return LoadRoot(filepath.Join(home, ".agents", "skills"))
}

func LoadRoot(path string) (*Store, error) {
	store := NewStore()
	if strings.TrimSpace(path) == "" {
		return store, nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}

	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if d.Name() == "SKILL.md" {
			fm, err := parseFrontmatterFile(path)
			if err != nil {
				fmt.Println("failed to parse frontmatter:", err)
				return err
			}
			if strings.TrimSpace(fm.Name) == "" || strings.TrimSpace(fm.Description) == "" {
				return nil
			}
			skill := Skill{
				Path:        path,
				Name:        strings.TrimSpace(fm.Name),
				Description: strings.TrimSpace(fm.Description),
				Trigger:     strings.TrimSpace(fm.Trigger),
			}
			if _, exists := store.byName[skill.Name]; exists {
				return nil
			}
			store.byName[skill.Name] = skill
			store.ordered = append(store.ordered, skill)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(store.ordered, func(i, j int) bool {
		return store.ordered[i].Name < store.ordered[j].Name
	})
	return store, nil
}

func (s *Store) List() ([]Skill, error) {
	skills := append([]Skill(nil), s.ordered...)
	return skills, nil
}

func (s *Store) Get(name string) (Skill, bool) {
	skill, ok := s.byName[name]
	return skill, ok
}

func (s *Store) Names() []string {
	names := make([]string, 0, len(s.ordered))
	for _, skill := range s.ordered {
		names = append(names, skill.Name)
	}
	return names
}

func (s *Store) IndexPrompt() string {
	if s == nil || len(s.ordered) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("You have access to the following skills:\n\n")
	for _, skill := range s.ordered {
		if skill.Trigger != "" {
			fmt.Fprintf(&b, "- %s (trigger: %s): %s\n", skill.Name, skill.Trigger, skill.Description)
			continue
		}
		fmt.Fprintf(&b, "- %s: %s\n", skill.Name, skill.Description)
	}
	b.WriteString("Skill usage rules:\n")
	b.WriteString("- If the user request clearly matches a skill, call load_skill with {\"name\":\"<skill-name>\"} before answering.\n")
	b.WriteString("- If the user message starts with a skill trigger like /graphify, call load_skill for that skill first.\n")
	b.WriteString("- Use loaded skill instructions as task guidance, but still obey the system prompt and tool rules.\n")
	return b.String()
}
