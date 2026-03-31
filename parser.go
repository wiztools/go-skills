package skills

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type frontMatter struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Metadata    map[string]any `yaml:"metadata"`
}

func loadSkill(rootDir, skillFile string) (*Skill, error) {
	raw, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", skillFile, err)
	}

	fmBytes, body, err := splitFrontMatter(raw)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", skillFile, err)
	}

	var fm frontMatter
	rawFM := map[string]any{}
	if len(fmBytes) > 0 {
		if err := yaml.Unmarshal(fmBytes, &fm); err != nil {
			return nil, fmt.Errorf("decode front matter in %q: %w", skillFile, err)
		}
		if err := yaml.Unmarshal(fmBytes, &rawFM); err != nil {
			return nil, fmt.Errorf("decode raw front matter in %q: %w", skillFile, err)
		}
	}

	skillDir := filepath.Dir(skillFile)
	files, err := listSkillFiles(skillDir)
	if err != nil {
		return nil, fmt.Errorf("list files for %q: %w", skillFile, err)
	}

	name := strings.TrimSpace(fm.Name)
	if name == "" {
		name = filepath.Base(skillDir)
	}

	return &Skill{
		Name:           name,
		Description:    strings.TrimSpace(fm.Description),
		Metadata:       cloneMap(fm.Metadata),
		RawFrontMatter: cloneMap(rawFM),
		RootDir:        rootDir,
		Dir:            skillDir,
		SkillFile:      skillFile,
		FilesInSkill:   files,
		markdown:       string(body),
	}, nil
}

func splitFrontMatter(raw []byte) ([]byte, []byte, error) {
	const delimiter = "---"

	if !bytes.HasPrefix(raw, []byte(delimiter+"\n")) && !bytes.Equal(raw, []byte(delimiter)) {
		return nil, raw, nil
	}

	lines := strings.Split(string(raw), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != delimiter {
		return nil, raw, nil
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == delimiter {
			fm := strings.Join(lines[1:i], "\n")
			body := strings.Join(lines[i+1:], "\n")
			body = strings.TrimPrefix(body, "\n")
			return []byte(fm), []byte(body), nil
		}
	}

	return nil, nil, fmt.Errorf("front matter start found without closing delimiter")
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
