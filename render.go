package skills

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Renderer converts a catalog to an LLM-consumable index representation.
type Renderer interface {
	Render(*Catalog) ([]byte, error)
}

// JSONRenderer renders the catalog as pretty-printed JSON.
type JSONRenderer struct {
	Indent string
}

// MarkdownRenderer renders the catalog as a compact markdown index.
type MarkdownRenderer struct{}

// RenderJSON renders the catalog as JSON using the default renderer settings.
func RenderJSON(c *Catalog) ([]byte, error) {
	return c.Render(JSONRenderer{Indent: "  "})
}

// RenderMarkdown renders the catalog as markdown using the default renderer.
func RenderMarkdown(c *Catalog) ([]byte, error) {
	return c.Render(MarkdownRenderer{})
}

func (r JSONRenderer) Render(c *Catalog) ([]byte, error) {
	indent := r.Indent
	if indent == "" {
		indent = "  "
	}

	payload := struct {
		Roots  []string `json:"roots"`
		Skills []*Skill `json:"skills"`
	}{
		Roots:  c.Roots(),
		Skills: c.Skills(),
	}

	return json.MarshalIndent(payload, "", indent)
}

func (MarkdownRenderer) Render(c *Catalog) ([]byte, error) {
	var b strings.Builder

	b.WriteString("# Skills Index\n\n")
	for _, skill := range c.Skills() {
		b.WriteString(fmt.Sprintf("## %s\n\n", skill.Name))
		if skill.Description != "" {
			b.WriteString(fmt.Sprintf("%s\n\n", skill.Description))
		}

		b.WriteString(fmt.Sprintf("- Directory: `%s`\n", skill.Dir))
		b.WriteString(fmt.Sprintf("- Skill file: `%s`\n", skill.SkillFile))
		if len(skill.Metadata) > 0 {
			b.WriteString("- Metadata:\n")
			for _, line := range renderMapLines(skill.Metadata) {
				b.WriteString(fmt.Sprintf("  - %s\n", line))
			}
		}
		if len(skill.FilesInSkill) > 0 {
			b.WriteString("- Files:\n")
			for _, file := range skill.FilesInSkill {
				b.WriteString(fmt.Sprintf("  - `%s`\n", file))
			}
		}
		b.WriteString("\n")
	}

	return []byte(b.String()), nil
}
