# AGENTS.md

## Purpose

This module loads AI skill directories into an in-memory catalog and renders that catalog into machine-friendly index formats.

Use it when you need to:

- discover skills from one or more parent directories
- inspect parsed front matter
- retrieve the markdown body of a specific skill
- enumerate files contained in a skill
- emit a catalog in JSON, Markdown, or another renderer-defined format

## Package Overview

- [`catalog.go`](/Users/subhash/code/bhq/go-skills/catalog.go): catalog construction, discovery, lookup, and rendering entry points
- [`skill.go`](/Users/subhash/code/bhq/go-skills/skill.go): in-memory skill model
- [`parser.go`](/Users/subhash/code/bhq/go-skills/parser.go): `SKILL.md` parsing and front matter extraction
- [`render.go`](/Users/subhash/code/bhq/go-skills/render.go): renderer interface and built-in JSON/Markdown renderers
- [`render_helpers.go`](/Users/subhash/code/bhq/go-skills/render_helpers.go): rendering helpers

## Main Types

### `Catalog`

Construct with:

```go
catalog, err := skills.NewCatalog(dir1, dir2, dir3)
```

Or separate creation from loading:

```go
catalog := skills.New(skills.Config{
	DuplicatePolicy: skills.DuplicateSkillSkip,
})
err := catalog.Load(dir1, dir2, dir3)
```

Key behavior:

- walks each provided parent directory recursively
- treats every `SKILL.md` as one skill
- loads all skills eagerly into memory
- indexes skills by resolved name
- applies configurable duplicate-name policy across roots

Key methods:

- `Roots() []string`
- `Skills() []*Skill`
- `Skill(name string) (*Skill, bool)`
- `MustSkill(name string) (*Skill, error)`
- `SkillMarkdown(name string) (string, error)`
- `Render(renderer Renderer) ([]byte, error)`
- `Load(skillDirs ...string) error`

### `Config`

Current fields:

- `DuplicatePolicy`
- `Logger`
- `Debug`

Duplicate policy options:

- `DuplicateSkillError`
- `DuplicateSkillSkip`
- `DuplicateSkillOverwrite`

Logging behavior:

- `Logger` is a `*slog.Logger`
- nil logger falls back to `slog.Default()`
- `Debug=true` enables detailed logs during `Load()`
- logs are intended to show root resolution, walking, skill discovery, duplicate handling, and final counts

### `Skill`

Important fields:

- `Name`
- `Description`
- `Metadata`
- `RawFrontMatter`
- `RootDir`
- `Dir`
- `SkillFile`
- `FilesInSkill`

Important methods:

- `Markdown() string`
- `Files() []string`

## Parsing Semantics

- Front matter is optional.
- Front matter must use `---` delimiters if present.
- `description` is parsed as a string.
- `metadata` is parsed as a generic `map[string]any`.
- `RawFrontMatter` preserves the decoded full front matter for downstream consumers.
- If `name` is absent, the skill directory basename becomes the skill name.

## Rendering Semantics

Built-in renderers:

- `JSONRenderer`
- `MarkdownRenderer`

Helper functions:

- `RenderJSON(catalog)`
- `RenderMarkdown(catalog)`

To add another output format, implement:

```go
type Renderer interface {
	Render(*Catalog) ([]byte, error)
}
```

## File Inventory

`FilesInSkill` and `Files()` expose the full file list for a skill directory.

Properties:

- paths are relative to the skill directory
- paths use slash separators
- `SKILL.md` is included
- ordering is deterministic

## Usage Guidance

- Prefer `MustSkill()` when absence is an error path you want reported.
- Prefer `Skill()` when probing optionally for a skill.
- Use `Markdown()` or `SkillMarkdown()` when you need only the body content for prompting.
- Use `RawFrontMatter` if new front matter keys need to be preserved without changing the struct API.

## Extensibility

Safe extension points:

- add new renderer types
- add richer typed metadata helpers on top of `Metadata`
- add filtering or search helpers on `Catalog`
- add serialization helpers that transform `Skill` into narrower views

Be careful when changing:

- duplicate name behavior
- front matter split logic
- file ordering or path normalization
- JSON field names on exported structs

## Validation

Before shipping changes, run:

```bash
go test ./...
```
