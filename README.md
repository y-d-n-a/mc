
# Multicoder - Go Edition

## Setup

1. Install Go 1.21 or later
2. Copy `example.env` to `.env` and configure values
3. Run `go mod download`
4. Build:
   ```bash
   chmod +x build.sh
   ./build.sh
   ```
5. Install aliases:
   ```bash
   chmod +x update_aliases.sh
   ./update_aliases.sh
   source ~/.zshrc
   ```

## Usage

### Multicoder (mc)
```bash
mc get 1 "*"              # Get all files in cwd, send to 1 LLM
mc get 1 -r "*"           # Get all files recursively, send to 1 LLM
mc get 3 -r "*.go"        # Get Go files recursively, send to 3 LLMs
mc get 1 main.go utils.go # Get specific files, send to 1 LLM
mc get 1 "*.go" -- fix the bug in the parser   # Inline instructions
mc write 0                # Write response 0 to disk
mc write list             # List available responses
mc open                   # View all responses
mc open 0                 # View response 0
mc checkpoint             # Set checkpoint at current version
mc rollback               # Rollback to latest version
mc rollback 5             # Rollback to version 5
mc rollback checkpoint    # Rollback to last checkpoint
mc clear                  # Clear workspace (with confirmation)
mc clear -y               # Clear workspace (no confirmation)
mc undo                   # Undo last write operation
mc ignore "*.log"         # Add ignore pattern
mc rmignore "*.log"       # Remove ignore pattern
mc lsignores              # List ignore patterns
mc model                  # Select model from saved models
mc model add              # Add models to saved list (OpenRouter or local)
mc model remove           # Remove models from saved list (multi-select)
mc repeat                 # Repeat last get call once
mc repeat 3               # Repeat last get call 3 times
mc prompt add <name>      # Add new system prompt
mc prompt delete <name>   # Delete system prompt
mc prompt update <name>   # Update existing prompt
mc prompt switch          # Switch active prompt (interactive)
mc prompt switch <name>   # Switch to specific prompt
mc prompt switch null     # Disable system prompt
mc prompt list            # List all prompts
mc cost                   # Show project cost summary
mc cost clear             # Clear project cost data
```

### Ask (single query)
```bash
ask "What is the capital of France?"
```

### Ask Chat (interactive)
```bash
askc
```

## Model Routing

Model routing is determined by the prefix of the model ID in your `.env` file.

### OpenRouter (cloud)

Any model ID that does not start with `ollama/` is routed to OpenRouter:

```
AI_TOOLS_MODEL=anthropic/claude-sonnet-4.6
ASK_APP_MODEL=google/gemini-3.1-pro-preview
```

Requires `OPENROUTER_API_KEY` and optionally `OPENROUTER_URL` (defaults to `https://openrouter.ai/api/v1`).

### Local cluster (ollama/)

Any model ID starting with `ollama/` is routed to your local cluster endpoint:

```
AI_TOOLS_MODEL=ollama/qwen3.5
ASK_APP_MODEL=ollama/llama3
```

Requires `LOCAL_URL` to be set in `.env`. The `ollama/` prefix is stripped before the
request is sent, so the local endpoint receives just `qwen3.5` or `llama3` as the
model name.

You can mix routing between the two env vars:

```
AI_TOOLS_MODEL=ollama/qwen3.5
ASK_APP_MODEL=anthropic/claude-sonnet-4.6
```

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `OPENROUTER_API_KEY` | For cloud models | — | OpenRouter API key |
| `OPENROUTER_URL` | No | `https://openrouter.ai/api/v1` | OpenRouter base URL |
| `LOCAL_URL` | For local models | — | Local cluster base URL (OpenAI-compatible) |
| `AI_TOOLS_MODEL` | Yes | — | Model for `mc get` |
| `ASK_APP_MODEL` | Yes | — | Model for `ask` / `askc` |
| `MAX_TOKENS` | No | `4096` | Max tokens per response |
| `TIMEOUT` | No | `120` | Request timeout in seconds |

## models.json

`models.json` is the local catalogue of models available for selection via `mc model`.

### Adding OpenRouter models

```bash
mc model add
# Select option 1: Add from OpenRouter API
# Use SPACE to select models, ENTER to confirm
```

### Adding local models

```bash
mc model add
# Select option 2: Add local model manually (ollama/...)
# Enter the model ID, e.g.: ollama/qwen3.5
```

Local model entries in `models.json` look like this:

```json
{
  "id": "ollama/qwen3.5",
  "name": "ollama/qwen3.5",
  "pricing": {
    "prompt": "0",
    "completion": "0",
    "request": "",
    "image": ""
  },
  "is_local": true
}
```

The `is_local: true` field marks the entry so the UI can label it `[local]` and so
the model list update logic does not attempt to refresh it from the OpenRouter API.

### Manually editing models.json

You can add a local model directly by appending an entry to `models.json`:

```json
{
  "id": "ollama/my-custom-model",
  "name": "ollama/my-custom-model",
  "pricing": {
    "prompt": "0",
    "completion": "0",
    "request": "",
    "image": ""
  },
  "is_local": true
}
```

Rules:
- `id` must start with `ollama/`
- `is_local` must be `true`
- `pricing` fields should be `"0"` (local models have no API cost)
- All other fields are optional

After adding manually, run `mc model` to select it.

## Checkpoint System

1. `mc checkpoint` — marks the current version as a known good state
2. `mc rollback checkpoint` — restores to that state

## System Prompts

Stored in `.sys_prompts/`. Managed with:

- `mc prompt add <name>` — create (opens editor)
- `mc prompt update <name>` — edit existing
- `mc prompt delete <name>` — remove
- `mc prompt switch` — interactive select
- `mc prompt switch null` — disable
- `mc prompt list` — view all

## Repeat

`mc repeat` re-executes the last `mc get` call with identical parameters.

- `mc repeat` — once
- `mc repeat 3` — three times sequentially

## Cost Tracking

All API calls are tracked per project.

- `mc cost` — view total cost and recent calls
- `mc cost clear` — reset tracking

Local (`ollama/`) models record zero cost. Cost data is stored in `.mcoder-workspace/.cost`.

## Ignore Patterns

`.mcignore` supports glob patterns, directory names, and substring matching.
Version control directories (`.git`, `.svn`, `.hg`, `.bzr`) are always ignored.

## File Gathering

`mc get` accepts one or more targets after the count:

- **Plain filename**: `mc get 1 main.go` — looks up `main.go` in cwd; with `-r`, searches all subdirectories by exact base name.
- **Glob pattern**: `mc get 1 "*.go"` — matches files in cwd; with `-r`, matches against the base filename of every file under cwd.
- **Multiple targets**: `mc get 1 main.go utils.go "*.md"` — all targets are resolved and deduplicated.
- **Relative path**: `mc get 1 cmd/mc/main.go` — treated as a direct path, no glob expansion.

Always quote glob patterns to prevent shell expansion: `mc get 1 '*.go'`

## Structure

- `pkg/config` — configuration loading
- `pkg/ai` — AI model interface, routing, cost tracking
- `pkg/multicoder` — multicoder functionality
- `cmd/ask` — single-query CLI
- `cmd/askc` — interactive chat CLI
- `cmd/mc` — multicoder CLI
