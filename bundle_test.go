package skills

import (
	"strings"
	"testing"
)

func TestCatalogSkillMarkdownBundle_IncludesRootAndReferencesInStableOrder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root+"/common/references/creative_principles.md", "[Scaffolds](copy_scaffolds.md)\ncreative")
	writeTestFile(t, root+"/common/references/copy_scaffolds.md", "scaffolds")
	writeTestFile(t, root+"/static-visual/references/prompt-templates.md", "templates")
	writeTestFile(t, root+"/static-visual/SKILL.md", strings.Join([]string{
		"---",
		"name: static-visual",
		"---",
		"# Static Visual",
		"[Creative](../common/references/creative_principles.md)",
		"[Templates](references/prompt-templates.md)",
		"[Creative Again](../common/references/creative_principles.md)",
	}, "\n"))

	catalog, err := NewCatalog(root)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	bundle, err := catalog.SkillMarkdownBundle("static-visual")
	if err != nil {
		t.Fatalf("SkillMarkdownBundle() error = %v", err)
	}

	if got, want := bundle.RootFile, "static-visual/SKILL.md"; got != want {
		t.Fatalf("RootFile = %q, want %q", got, want)
	}

	gotPaths := []string{}
	for _, file := range bundle.Files {
		gotPaths = append(gotPaths, file.Path)
	}
	wantPaths := []string{
		"static-visual/SKILL.md",
		"common/references/creative_principles.md",
		"static-visual/references/prompt-templates.md",
		"common/references/copy_scaffolds.md",
	}
	if !equalStrings(gotPaths, wantPaths) {
		t.Fatalf("bundle file paths = %v, want %v", gotPaths, wantPaths)
	}

	if !strings.Contains(bundle.Markdown, "## ROOT SKILL FILE: static-visual/SKILL.md") {
		t.Fatalf("bundle markdown missing root header: %s", bundle.Markdown)
	}
	if !strings.Contains(bundle.Markdown, "## REFERENCED FILE: common/references/creative_principles.md") {
		t.Fatalf("bundle markdown missing creative principles header: %s", bundle.Markdown)
	}
	if strings.Count(bundle.Markdown, "## REFERENCED FILE: common/references/creative_principles.md") != 1 {
		t.Fatalf("bundle markdown duplicated creative principles section: %s", bundle.Markdown)
	}
}

func TestSkillMarkdownBundle_SkipsExternalAndNonMarkdownLinks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root+"/alpha/references/local.md", "local")
	writeTestFile(t, root+"/alpha/SKILL.md", strings.Join([]string{
		"# Alpha",
		"[Local](references/local.md)",
		"[External](https://example.com/guide.md)",
		"[Image](references/diagram.png)",
		"[Anchor](references/local.md#section)",
	}, "\n"))

	catalog, err := NewCatalog(root)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	bundle, err := catalog.SkillMarkdownBundle("alpha")
	if err != nil {
		t.Fatalf("SkillMarkdownBundle() error = %v", err)
	}

	if got, want := len(bundle.Files), 2; got != want {
		t.Fatalf("len(bundle.Files) = %d, want %d", got, want)
	}
	if bundle.Files[1].Path != "alpha/references/local.md" {
		t.Fatalf("bundle second file path = %q, want %q", bundle.Files[1].Path, "alpha/references/local.md")
	}
}

func TestSkillMarkdownBundle_MissingReferenceReturnsError(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root+"/alpha/SKILL.md", strings.Join([]string{
		"# Alpha",
		"[Missing](references/missing.md)",
	}, "\n"))

	catalog, err := NewCatalog(root)
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	_, err = catalog.SkillMarkdownBundle("alpha")
	if err == nil || !strings.Contains(err.Error(), "missing.md") {
		t.Fatalf("SkillMarkdownBundle() error = %v, want missing reference error", err)
	}
}
