package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	errBundlePathEscapesRoot = errors.New("bundle path escapes root")
	errBundleFileMissing     = errors.New("bundle file not found")
	markdownLinkPattern      = regexp.MustCompile(`\[[^\]]+\]\(([^)#]+)(?:#[^)]+)?\)`)
)

// BundledFile is one file included in a markdown bundle.
type BundledFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// MarkdownBundle contains a root skill file plus its linked markdown references
// resolved under the same root directory.
type MarkdownBundle struct {
	RootFile string        `json:"root_file"`
	Files    []BundledFile `json:"files"`
	Markdown string        `json:"markdown"`
}

// SkillMarkdownBundle returns a bundled markdown context for a named skill.
// The bundle contains the root SKILL.md plus linked markdown references in
// stable, deduplicated order.
func (c *Catalog) SkillMarkdownBundle(name string) (*MarkdownBundle, error) {
	skill, err := c.MustSkill(name)
	if err != nil {
		return nil, err
	}
	return skill.MarkdownBundle()
}

// MarkdownBundle returns a bundled markdown context for this skill. The bundle
// contains the root SKILL.md plus linked markdown references in stable,
// deduplicated order.
func (s *Skill) MarkdownBundle() (*MarkdownBundle, error) {
	rootFile, err := filepath.Rel(s.RootDir, s.SkillFile)
	if err != nil {
		return nil, fmt.Errorf("resolve root skill file %q: %w", s.SkillFile, err)
	}
	return buildMarkdownBundle(s.RootDir, filepath.ToSlash(rootFile))
}

func buildMarkdownBundle(rootDir, rootFile string) (*MarkdownBundle, error) {
	rootRel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rootFile)))
	queue := []string{rootRel}
	seen := map[string]bool{}
	bundle := &MarkdownBundle{RootFile: rootRel}
	var sb strings.Builder

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if seen[current] {
			continue
		}
		seen[current] = true

		absCurrent, currentRel, err := resolveBundlePath(rootDir, current, current)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", current, err)
		}

		data, err := os.ReadFile(absCurrent)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", currentRel, err)
		}

		if sb.Len() > 0 {
			sb.WriteString("\n\n---\n\n")
		}
		if len(bundle.Files) == 0 {
			sb.WriteString(fmt.Sprintf("## ROOT SKILL FILE: %s\n\n", currentRel))
		} else {
			sb.WriteString(fmt.Sprintf("## REFERENCED FILE: %s\n\n", currentRel))
		}

		content := string(data)
		sb.WriteString(content)
		bundle.Files = append(bundle.Files, BundledFile{
			Path:    currentRel,
			Content: content,
		})

		for _, ref := range extractMarkdownRefs(content) {
			_, nextRel, err := resolveBundlePath(rootDir, currentRel, ref)
			if err != nil {
				return nil, fmt.Errorf("resolve %q from %s: %w", ref, currentRel, err)
			}
			if !seen[nextRel] {
				queue = append(queue, nextRel)
			}
		}
	}

	bundle.Markdown = sb.String()
	return bundle, nil
}

func extractMarkdownRefs(markdown string) []string {
	matches := markdownLinkPattern.FindAllStringSubmatch(markdown, -1)
	seen := map[string]bool{}
	refs := []string{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		ref := strings.TrimSpace(match[1])
		if ref == "" {
			continue
		}
		lower := strings.ToLower(ref)
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
			continue
		}
		if filepath.Ext(lower) != ".md" {
			continue
		}
		if seen[ref] {
			continue
		}
		seen[ref] = true
		refs = append(refs, ref)
	}
	return refs
}

func resolveBundlePath(rootDir, currentFile, ref string) (absTarget string, rel string, err error) {
	currentRel := filepath.Clean(filepath.FromSlash(currentFile))
	refRel := filepath.Clean(filepath.FromSlash(ref))
	absCurrentFile := filepath.Join(rootDir, currentRel)

	candidates := []string{}
	if refRel == currentRel {
		candidates = append(candidates, absCurrentFile)
	}
	candidates = append(candidates, filepath.Clean(filepath.Join(filepath.Dir(absCurrentFile), refRel)))
	if strings.Contains(ref, "/") || strings.Contains(ref, "\\") {
		candidates = append(candidates, filepath.Clean(filepath.Join(rootDir, refRel)))
	}

	var firstSafeRel string
	for _, candidate := range candidates {
		candidateRel, relErr := filepath.Rel(rootDir, candidate)
		if relErr != nil || strings.HasPrefix(candidateRel, "..") {
			continue
		}
		normalizedRel := filepath.ToSlash(candidateRel)
		if firstSafeRel == "" {
			firstSafeRel = normalizedRel
		}
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, normalizedRel, nil
		}
	}

	if firstSafeRel == "" {
		return "", "", errBundlePathEscapesRoot
	}
	return "", firstSafeRel, errBundleFileMissing
}
