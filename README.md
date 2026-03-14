# github-mcp

A GitHub MCP (Model Context Protocol) server written in Go that lets AI assistants like Claude interact with your GitHub account.

## Tools

| Tool | Description |
|------|-------------|
| `gitgub_list_repos` | List all repositories for the authenticated user |
| `github_create_repo` | Create a new repository |
| `github_delete_repo` | Permanently delete a repository |
| `github_update_visibility` | Change a repository between public and private |
| `github_trigger_workflow` | Trigger a GitHub Actions workflow |
| `github_list_workflow_runs` | List recent workflow runs for a repository |
| `github_get_repo` | Get detailed info about a specific repository |

## Prerequisites

- [Go](https://golang.org/dl/) 1.25+
- A GitHub Personal Access Token with the following scopes:
  - `repo`
  - `delete_repo`
  - `workflow`

Create one at: https://github.com/settings/tokens

## Install

```bash
go install github.com/Stevke7/github-mcp@latest
```

This will install the binary to your `$GOPATH/bin` directory.

## Build from source

```bash
git clone https://github.com/Stevke7/github-mcp.git
cd github-mcp
go build -o github-mcp
```

## Configuration

Set your GitHub token as an environment variable:

```bash
export GITHUB_TOKEN=your_token_here
```

## Usage with Claude Code

Add the server to your Claude Code MCP configuration (`~/.claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "github-mcp": {
      "command": "/path/to/github-mcp",
      "env": {
        "GITHUB_TOKEN": "your_token_here"
      }
    }
  }
}
```

Then restart Claude Code. You can now ask Claude to manage your GitHub repositories directly.

## Example Prompts

- *"List all my repositories"*
- *"Create a new private repo called my-project"*
- *"Change my-repo to private"*
- *"Trigger the CI workflow on the main branch of my-repo"*
- *"Show me the last 5 workflow runs for my-repo"*
