package skills

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
)

// DuplicateSkillPolicy controls how duplicate skill names are handled while
// loading a catalog.
type DuplicateSkillPolicy int

const (
	// DuplicateSkillError aborts loading when a duplicate skill name is found.
	DuplicateSkillError DuplicateSkillPolicy = iota
	// DuplicateSkillSkip keeps the first discovered skill and ignores later
	// duplicates.
	DuplicateSkillSkip
	// DuplicateSkillOverwrite keeps the most recently discovered skill.
	DuplicateSkillOverwrite
)

// Config controls catalog loading behavior.
type Config struct {
	DuplicatePolicy DuplicateSkillPolicy
	Logger          *slog.Logger
	Debug           bool
}

// DefaultConfig returns the default catalog configuration.
func DefaultConfig() Config {
	return Config{
		DuplicatePolicy: DuplicateSkillError,
	}
}

// Catalog holds a loaded in-memory index of skills discovered from one or more
// root directories.
type Catalog struct {
	config Config
	roots  []string
	skills []*Skill
	byName map[string]*Skill
}

// NewCatalog discovers and loads every skill found under the provided skill
// root directories.
func NewCatalog(skillDirs ...string) (*Catalog, error) {
	return NewCatalogWithConfig(DefaultConfig(), skillDirs...)
}

// NewCatalogWithConfig creates a catalog with the provided configuration and
// eagerly loads the supplied roots.
func NewCatalogWithConfig(config Config, skillDirs ...string) (*Catalog, error) {
	c := New(config)
	if err := c.Load(skillDirs...); err != nil {
		return nil, err
	}
	return c, nil
}

// New creates an empty catalog with the provided configuration. Use Load to
// discover skills from one or more roots.
func New(config Config) *Catalog {
	if !config.DuplicatePolicy.valid() {
		config.DuplicatePolicy = DuplicateSkillError
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	return &Catalog{
		config: config,
		roots:  []string{},
		skills: []*Skill{},
		byName: map[string]*Skill{},
	}
}

// Load discovers and loads every skill found under the provided skill root
// directories.
func (c *Catalog) Load(skillDirs ...string) error {
	if len(skillDirs) == 0 {
		return errors.New("at least one skills directory is required")
	}

	c.debug("starting skill catalog load", "roots", skillDirs)
	for _, root := range skillDirs {
		if err := c.loadRoot(root); err != nil {
			c.debug("skill catalog load failed", "root", root, "error", err)
			return err
		}
	}

	c.rebuildSkillsSlice()
	c.debug("completed skill catalog load", "root_count", len(c.roots), "skill_count", len(c.skills))

	return nil
}

// Roots returns the normalized skill roots used to build the catalog.
func (c *Catalog) Roots() []string {
	return append([]string(nil), c.roots...)
}

// Skills returns the loaded skills in deterministic order.
func (c *Catalog) Skills() []*Skill {
	return append([]*Skill(nil), c.skills...)
}

// Skill looks up a skill by its resolved name.
func (c *Catalog) Skill(name string) (*Skill, bool) {
	skill, ok := c.byName[name]
	return skill, ok
}

// MustSkill returns a skill by name or an error when the skill is absent.
func (c *Catalog) MustSkill(name string) (*Skill, error) {
	skill, ok := c.Skill(name)
	if !ok {
		return nil, fmt.Errorf("skill %q not found", name)
	}
	return skill, nil
}

// SkillMarkdown returns the markdown body of a named skill's SKILL.md.
func (c *Catalog) SkillMarkdown(name string) (string, error) {
	skill, err := c.MustSkill(name)
	if err != nil {
		return "", err
	}
	return skill.Markdown(), nil
}

// Render renders the catalog with the supplied renderer.
func (c *Catalog) Render(renderer Renderer) ([]byte, error) {
	if renderer == nil {
		return nil, errors.New("renderer is required")
	}
	return renderer.Render(c)
}

func (c *Catalog) loadRoot(root string) error {
	c.debug("resolving skills root", "input_root", root)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve skills root %q: %w", root, err)
	}

	c.debug("checking skills root", "root", absRoot)
	info, err := os.Stat(absRoot)
	if err != nil {
		return fmt.Errorf("stat skills root %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("skills root %q is not a directory", absRoot)
	}

	c.roots = append(c.roots, absRoot)
	c.debug("walking skills root", "root", absRoot)

	return filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "SKILL.md" {
			return nil
		}

		c.debug("discovered skill file", "skill_file", path)
		skill, err := loadSkill(absRoot, path)
		if err != nil {
			return err
		}
		c.debug("loaded skill metadata", "name", skill.Name, "dir", skill.Dir, "file_count", len(skill.FilesInSkill))
		if existing, exists := c.byName[skill.Name]; exists {
			switch c.config.DuplicatePolicy {
			case DuplicateSkillSkip:
				c.debug("skipping duplicate skill", "name", skill.Name, "existing_dir", existing.Dir, "duplicate_dir", skill.Dir)
				return nil
			case DuplicateSkillOverwrite:
				c.debug("overwriting duplicate skill", "name", skill.Name, "previous_dir", existing.Dir, "new_dir", skill.Dir)
			case DuplicateSkillError:
				fallthrough
			default:
				c.debug("duplicate skill caused error", "name", skill.Name, "existing_dir", existing.Dir, "duplicate_dir", skill.Dir)
				return fmt.Errorf("duplicate skill name %q discovered at %q", skill.Name, path)
			}
		}

		c.byName[skill.Name] = skill
		c.debug("registered skill", "name", skill.Name, "dir", skill.Dir)
		return nil
	})
}

func listSkillFiles(skillDir string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(skillDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func (c *Catalog) rebuildSkillsSlice() {
	c.skills = c.skills[:0]
	for _, skill := range c.byName {
		c.skills = append(c.skills, skill)
	}
	sort.Slice(c.skills, func(i, j int) bool {
		return c.skills[i].Name < c.skills[j].Name
	})
}

func (p DuplicateSkillPolicy) valid() bool {
	switch p {
	case DuplicateSkillError, DuplicateSkillSkip, DuplicateSkillOverwrite:
		return true
	default:
		return false
	}
}

func (c *Catalog) debug(msg string, args ...any) {
	if !c.config.Debug {
		return
	}
	c.config.Logger.Debug(msg, args...)
}
