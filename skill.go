package skills

// Skill contains the parsed metadata and content for one skill directory.
type Skill struct {
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	RawFrontMatter map[string]any `json:"raw_front_matter,omitempty"`
	RootDir        string         `json:"root_dir"`
	Dir            string         `json:"dir"`
	SkillFile      string         `json:"skill_file"`
	FilesInSkill   []string       `json:"files"`

	markdown string
}

// Markdown returns the markdown body of SKILL.md, excluding YAML front matter.
func (s *Skill) Markdown() string {
	return s.markdown
}

// Files returns the files contained in the skill directory.
func (s *Skill) Files() []string {
	return append([]string(nil), s.FilesInSkill...)
}
