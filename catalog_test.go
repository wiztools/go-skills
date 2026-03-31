package skills

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogLoadsSkillsAcrossRoots(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()

	alphaDir := filepath.Join(rootA, "alpha")
	writeTestFile(t, filepath.Join(alphaDir, "SKILL.md"), `---
name: alpha
description: Alpha skill
metadata:
  category: utility
  score: 7
---
# Alpha

Alpha body.
`)
	writeTestFile(t, filepath.Join(alphaDir, "references", "notes.md"), "reference")

	betaDir := filepath.Join(rootB, "group", "beta")
	writeTestFile(t, filepath.Join(betaDir, "SKILL.md"), `---
description: Beta skill
---
# Beta

Beta body.
`)
	writeTestFile(t, filepath.Join(betaDir, "scripts", "run.sh"), "#!/bin/sh")

	catalog, err := NewCatalog(rootA, rootB)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	if got, want := len(catalog.Skills()), 2; got != want {
		t.Fatalf("len(Skills()) = %d, want %d", got, want)
	}

	alpha, ok := catalog.Skill("alpha")
	if !ok {
		t.Fatalf("alpha skill not found")
	}
	if alpha.Description != "Alpha skill" {
		t.Fatalf("alpha description = %q", alpha.Description)
	}
	if alpha.Metadata["category"] != "utility" {
		t.Fatalf("alpha metadata category = %v", alpha.Metadata["category"])
	}
	if !strings.Contains(alpha.Markdown(), "Alpha body.") {
		t.Fatalf("alpha markdown = %q", alpha.Markdown())
	}
	if got, want := alpha.Files(), []string{"SKILL.md", "references/notes.md"}; !equalStrings(got, want) {
		t.Fatalf("alpha files = %v, want %v", got, want)
	}

	beta, ok := catalog.Skill("beta")
	if !ok {
		t.Fatalf("beta skill not found")
	}
	if beta.Name != "beta" {
		t.Fatalf("beta name = %q", beta.Name)
	}
	if !strings.Contains(beta.Markdown(), "# Beta") {
		t.Fatalf("beta markdown = %q", beta.Markdown())
	}
}

func TestRenderers(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "alpha", "SKILL.md"), `---
name: alpha
description: Alpha skill
metadata:
  category: utility
---
# Alpha
`)

	catalog, err := NewCatalog(root)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	markdown, err := RenderMarkdown(catalog)
	if err != nil {
		t.Fatalf("RenderMarkdown() error = %v", err)
	}
	if !strings.Contains(string(markdown), "## alpha") {
		t.Fatalf("markdown index missing skill heading: %s", markdown)
	}

	jsonPayload, err := RenderJSON(catalog)
	if err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}
	if !strings.Contains(string(jsonPayload), `"name": "alpha"`) {
		t.Fatalf("json index missing skill name: %s", jsonPayload)
	}
}

func TestDuplicateSkillNamesFail(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()

	writeTestFile(t, filepath.Join(rootA, "one", "SKILL.md"), `---
name: duplicate
---
`)
	writeTestFile(t, filepath.Join(rootB, "two", "SKILL.md"), `---
name: duplicate
---
`)

	_, err := NewCatalog(rootA, rootB)
	if err == nil || !strings.Contains(err.Error(), "duplicate skill name") {
		t.Fatalf("NewCatalog() error = %v, want duplicate skill name error", err)
	}
}

func TestDuplicateSkillNamesCanBeSkipped(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()

	writeTestFile(t, filepath.Join(rootA, "one", "SKILL.md"), `---
name: duplicate
description: first
---
# First
`)
	writeTestFile(t, filepath.Join(rootB, "two", "SKILL.md"), `---
name: duplicate
description: second
---
# Second
`)

	catalog, err := NewCatalogWithConfig(Config{
		DuplicatePolicy: DuplicateSkillSkip,
	}, rootA, rootB)
	if err != nil {
		t.Fatalf("NewCatalogWithConfig() error = %v", err)
	}

	skill, ok := catalog.Skill("duplicate")
	if !ok {
		t.Fatalf("duplicate skill not found")
	}
	if skill.Description != "first" {
		t.Fatalf("description = %q, want first", skill.Description)
	}
}

func TestDuplicateSkillNamesCanBeOverwritten(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()

	writeTestFile(t, filepath.Join(rootA, "one", "SKILL.md"), `---
name: duplicate
description: first
---
# First
`)
	writeTestFile(t, filepath.Join(rootB, "two", "SKILL.md"), `---
name: duplicate
description: second
---
# Second
`)

	catalog, err := NewCatalogWithConfig(Config{
		DuplicatePolicy: DuplicateSkillOverwrite,
	}, rootA, rootB)
	if err != nil {
		t.Fatalf("NewCatalogWithConfig() error = %v", err)
	}

	skill, ok := catalog.Skill("duplicate")
	if !ok {
		t.Fatalf("duplicate skill not found")
	}
	if skill.Description != "second" {
		t.Fatalf("description = %q, want second", skill.Description)
	}
	if !strings.Contains(skill.Markdown(), "Second") {
		t.Fatalf("markdown = %q, want second body", skill.Markdown())
	}
}

func TestDebugLoggingUsesConfiguredLogger(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "alpha", "SKILL.md"), `---
name: alpha
description: Alpha skill
---
# Alpha
`)

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	catalog := New(Config{
		DuplicatePolicy: DuplicateSkillError,
		Logger:          logger,
		Debug:           true,
	})

	if err := catalog.Load(root); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "starting skill catalog load") {
		t.Fatalf("expected start log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "discovered skill file") {
		t.Fatalf("expected discovery log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "completed skill catalog load") {
		t.Fatalf("expected completion log, got %q", logOutput)
	}
}
