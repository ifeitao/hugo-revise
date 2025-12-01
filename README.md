# hugo-revise

Language: English | 中文版见 [README.zh-CN.md](README.zh-CN.md)

![Release](https://img.shields.io/github/v/release/your-username/hugo-revise)
![Go Version](https://img.shields.io/badge/go-1.22%2B-00ADD8?logo=go)
![Build](https://img.shields.io/github/actions/workflow/status/your-username/hugo-revise/ci.yml?label=build)

Minimal, practical Hugo content revision CLI with version history support.

## Features

- **Major revision tracking**: Designed for significant content revisions or rewrites, not a replacement for Git
- Supports single files (`.md`) and page bundles (`index.md`)
- Stores history in independent `.revisions` directories, avoiding nested bundle limitations
- Accurate URL detection via `hugo list all`, respecting permalink rules
- Date-based versioning (one revision per day maximum)
- Archived versions are not listed but are directly accessible (`build.list: never, render: true`)
- Simple `undo` to revert the last revision

## Installation

### From GitHub (Recommended)

```sh
go install github.com/your-username/hugo-revise/cmd/hugo-revise@latest
```

### From Source

```sh
git clone https://github.com/your-username/hugo-revise.git
cd hugo-revise
go install ./cmd/hugo-revise
```

## Usage

### Basic

```sh
# Create a revision for a single .md file
hugo-revise content/posts/my-post.md

# Or without .md extension (auto-detection)
hugo-revise content/posts/my-post

# Create a revision for a page bundle
hugo-revise content/posts/my-bundle

# Explicit subcommand
hugo-revise revise content/posts/my-post
```

### Undo

```sh
# Undo the last revision
hugo-revise undo
```

### Directory Structure

Single file:
```
content/posts/
├── my-post.md
└── my-post.revisions/
    ├── 2025-11-30.md
    └── 2025-12-01.md
```

Page bundle:
```
content/posts/
├── my-bundle/
│   ├── index.md
│   └── image.png
└── my-bundle.revisions/
    ├── 2025-11-30/
    │   ├── index.md
    │   └── image.png
    └── 2025-12-01/
        └── index.md
```

### Generated URLs

Archived versions automatically include `/revisions/` in URL:

- Current page: `/my-post/`
- Archive (Nov 30): `/my-post/revisions/2025-11-30/`
- Archive (Dec 1): `/my-post/revisions/2025-12-01/`

## Config `.hugo-reviserc.toml`

Place in your Hugo project root to customize date format:

```toml
[versioning]
date_format = "2006-01-02"  # Default format, customize as needed
```

**Note**: Only date-based versioning is supported. The date format follows Go's time formatting convention.

## Front Matter

### Current Version

```yaml
---
title: My Post
date: 2025-12-01T00:10:08+08:00       # Updated to current time (revision date)
lastmod: 2025-12-01T00:10:08+08:00   # Updated to current time
revisions_history:                   # List of all versions (chronologically sorted)
  - 2024-06-15
  - 2025-12-01
---
```

### Archived Version

```yaml
---
title: My Post
date: 2024-06-15                     # Original date preserved
lastmod: 2024-06-15T10:30:00+08:00   # Original lastmod preserved
url: "/my-post/revisions/2024-06-15/"  # Fixed URL for this archived version
build:
  list: never                        # Not shown in list pages
  render: true                       # But can be accessed directly
revisions_history:                   # Same list as current version
  - 2024-06-15
  - 2025-12-01
---
```

## URL Resolution Priority

1. Existing `url` field in front matter
2. `hugo list all` (respects permalink configuration)
3. `slug` + section derivation
4. Path-based fallback (e.g., `content/posts/demo` → `/posts/demo/`)

## Hugo Integration

### Show Revision History

Optionally copy `templates/layouts/partials/revision-history.html` to your Hugo project:

```sh
mkdir -p your-hugo-project/layouts/partials
cp templates/layouts/partials/revision-history.html \
   your-hugo-project/layouts/partials/
```

Reference in your post template:

```go-html-template
{{ partial "revision-history.html" . }}
```

## Notes

- **Tool purpose**: hugo-revise is for tracking major content revisions (rewrites, significant updates), not for daily edits. Use Git for granular version control.
- **One revision per day**: If you attempt to create multiple revisions on the same day, you'll receive an error. This is by design.
- Commit the working tree before revising: `git add -A && git commit -m "before revision"`
- Requires Hugo CLI available in PATH
- Run in the Hugo project root so config is found
- Hugo 0.145+: uses `build` field instead of deprecated `_build`
- Front matter field behavior:
  - `date`: Updated to current time in the current version (represents revision date); preserved in archived versions
  - `lastmod`: Updated to current time in the current version; preserved in archived versions
  - `revisions_history`: Added to both current and archived versions, contains chronologically sorted list of all version dates
  - `url`: Added to archived versions only, ensures stable permalink
  - `build`: Added to archived versions only, prevents them from appearing in list pages
  - Version labels are based on the revision date (one revision per day maximum)

## Roadmap

### M1 - Basics (Done)
- Single file and bundle support
- `.revisions` independent structure
- URL detection via `hugo list all`
- Version conflict handling
- Basic undo

### M2 - Enhancements (Planned)
- Support for custom version labels (e.g., v1, v2, v3)
- Enhanced template examples with better styling and features

---

For the Chinese version, see `README.zh-CN.md`.