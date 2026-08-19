---
title: For AI Agents
weight: 85
description: Machine-readable endpoints for consuming the KICK documentation with an LLM or agent.
llmsDescription: |
  Machine-readable documentation endpoints for KICK. llms.txt at the site root
  lists every page with a one-line summary. llms-full.txt contains the whole
  documentation set in one file. Every page is also served as clean Markdown:
  leaf pages at {page}.md, sections at {section}/index.md. AGENTS.md in the repo
  root instructs IDE coding agents. Regenerate with make generate.
---

KICK's documentation is published in machine-readable form so an agent can load
the whole project context in one or two requests.

## Endpoints

| URL | Content | Use case |
|-----|---------|----------|
| [`/kick/llms.txt`](/kick/llms.txt) | Compact project summary: APIs, runtime model, key references | Cheap orientation |
| [`/kick/llms-full.txt`](/kick/llms-full.txt) | Every documentation page concatenated into one file | One GET = full project context |
| `{page}.md` | Clean Markdown for a single page | Fetch one topic |

## Markdown output

Every page is served as Markdown next to its HTML. Leaf pages append `.md` to
the page path; sections append `index.md`:

```
/kick/docs/installation/           → HTML
/kick/docs/installation.md         → Markdown

/kick/docs/reference/              → HTML
/kick/docs/reference/index.md      → Markdown
```

## Context menu

Every documentation page has a context menu in the top-right with **Open in
ChatGPT** and **Open in Claude**. Both pre-load the page's Markdown URL, so the
model reads the page directly instead of relying on training data.

## IDE coding agents

`AGENTS.md` in the repository root is auto-discovered by IDE coding agents. It
carries the rules that matter when changing code: the required task workflow,
framework constraints, design constraints, traceability requirements, and the
local `make` targets.

## Regenerating

```bash
make generate
```

`hack/gen-docs.sh` rewrites `llms.txt`, rebuilds `llms-full.txt` from every
Markdown file under `docs/content/docs/`, and publishes a copy to
`docs/static/` so the site serves it. `make docs-gen-check` fails the build when
the generated files drift from the sources.
