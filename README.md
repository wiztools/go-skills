# go-skills

`go-skills` is a Go module for loading AI skill directories into memory and rendering them into LLM-friendly index formats.

It is designed for skill layouts where each skill lives in its own directory and is defined by a `SKILL.md` file with optional YAML front matter.

## What It Does

- Loads skills from one or more root directories
- Parses YAML front matter from each `SKILL.md`
- Keeps skill metadata in memory for lookup and rendering
- Separates front matter from the markdown body of `SKILL.md`
- Tracks the list of files contained in each skill directory
- Renders a catalog as JSON or Markdown
- Supports custom renderers for additional output formats

## Installation

```bash
go get github.com/wiztools/go-skills
```

## Expected Skill Layout

```text
skills-root/
  my-skill/
    SKILL.md
    references/commands.md
    scripts/helper.py
```

Example `SKILL.md`:

```md
---
name: my-skill
description: A sample skill
metadata:
  category: utility
  owner: platform
---

# My Skill

This is the markdown body of the skill.
```

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	skills "github.com/wiztools/go-skills"
)

func main() {
	catalog, err := skills.NewCatalog(
		"/Users/me/.codex/skills",
		"/path/to/project/.agents/skills",
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("loaded %d skills\n", len(catalog.Skills()))

	skill, err := catalog.MustSkill("my-skill")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("description:", skill.Description)
	fmt.Println("files:", skill.Files())
	fmt.Println("markdown body:")
	fmt.Println(skill.Markdown())
}
```

## Core API

### Catalog

Create a catalog from one or more parent directories:

```go
catalog, err := skills.NewCatalog("/parent/one", "/parent/two")
```

Or create an instance with configuration and load roots later:

```go
logger := slog.Default()

catalog := skills.New(skills.Config{
	DuplicatePolicy: skills.DuplicateSkillOverwrite,
	Logger:          logger,
	Debug:           true,
})

err := catalog.Load("/parent/one", "/parent/two")
```

`Load()` expects parent directories, not individual skill directories. It walks each parent recursively and treats every discovered `SKILL.md` as one skill.

Useful methods:

- `Roots() []string`
- `Skills() []*Skill`
- `Skill(name string) (*Skill, bool)`
- `MustSkill(name string) (*Skill, error)`
- `SkillMarkdown(name string) (string, error)`
- `SkillMarkdownBundle(name string) (*MarkdownBundle, error)`
- `Render(renderer Renderer) ([]byte, error)`
- `Load(skillDirs ...string) error`

### Configuration

Current `Config` fields:

- `DuplicatePolicy`
- `Logger`
- `Debug`

Behavior:

- `Logger` accepts a custom `*slog.Logger`
- if `Logger` is nil, the package uses `slog.Default()`
- when `Debug` is `true`, `Load()` emits debug logs describing root resolution, discovery, duplicate handling, registration, and completion

### Skill

Each `Skill` includes:

- `Name`
- `Description`
- `Metadata`
- `RawFrontMatter`
- `RootDir`
- `Dir`
- `SkillFile`
- `FilesInSkill`

Useful methods:

- `Markdown() string`
- `Files() []string`
- `MarkdownBundle() (*MarkdownBundle, error)`

### Markdown bundles

When a skill links to markdown references under the same root, you can build a
single bundled context that includes the root `SKILL.md` plus linked markdown
references in stable, deduplicated order.

```go
bundle, err := catalog.SkillMarkdownBundle("my-skill")
if err != nil {
	log.Fatal(err)
}

fmt.Println(bundle.RootFile)
fmt.Println(bundle.Markdown)
```

`MarkdownBundle` includes:

- `RootFile` — root-relative path to the starting `SKILL.md`
- `Files` — included files with root-relative paths and raw contents
- `Markdown` — one stitched markdown payload with explicit file boundaries

## Rendering

### Markdown index

```go
payload, err := skills.RenderMarkdown(catalog)
if err != nil {
	log.Fatal(err)
}
fmt.Println(string(payload))
```

### JSON index

```go
payload, err := skills.RenderJSON(catalog)
if err != nil {
	log.Fatal(err)
}
fmt.Println(string(payload))
```

## Custom Renderers

To support another format, implement the `Renderer` interface:

```go
type Renderer interface {
	Render(*Catalog) ([]byte, error)
}
```

Example:

```go
type PlainTextRenderer struct{}

func (PlainTextRenderer) Render(c *skills.Catalog) ([]byte, error) {
	return []byte("custom output"), nil
}
```

Then call:

```go
payload, err := catalog.Render(PlainTextRenderer{})
```

## Notes

- If a skill front matter does not define `name`, the directory basename is used.
- Duplicate skill handling is configurable:
  - `DuplicateSkillError` returns an error
  - `DuplicateSkillSkip` keeps the first discovered skill
  - `DuplicateSkillOverwrite` keeps the most recently discovered skill
- The markdown body returned by `Markdown()` excludes the YAML front matter.
- The file list is relative to the skill directory and includes `SKILL.md`.

## Development

Run tests with:

```bash
go test ./...
```
