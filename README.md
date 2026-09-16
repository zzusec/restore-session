<div align="center">
  <h1>Restore-session</h1>
  <p><strong>Browse your Codex, Claude Code, OpenCode, and Pi sessions from one terminal — and jump straight back into any of them with one key.</strong></p>
  <p>
    <a href="https://github.com/zzusec/restore-session/releases/latest"><img src="https://img.shields.io/github/v/release/zzusec/restore-session?label=release" alt="Latest release"></a>
    <a href="https://github.com/zzusec/restore-session/actions/workflows/ci.yml"><img src="https://github.com/zzusec/restore-session/actions/workflows/ci.yml/badge.svg" alt="CI status"></a>
    <img src="https://img.shields.io/badge/platforms-macOS%20%7C%20Linux%20%7C%20Windows-blue" alt="Platforms: macOS, Linux, and Windows">
    <a href="./LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="MIT License"></a>
  </p>
  <p><strong>English</strong> · <a href="./README.zh-CN.md">简体中文</a></p>
</div>

> **Fork notice &amp; acknowledgements**
>
> `restore-session` is a fork of [**agent-session-cleaner**](https://github.com/haowang02/agent-session-cleaner) by [**Hao Wang (haowang02)**](https://github.com/haowang02). All of the session browsing, search, cleanup, and archiving machinery is his work — this project would not exist without it. 🙏
>
> The one thing `restore-session` adds is a **one-key resume**: highlight a session in the browser, press `Enter`, and the current process is replaced by `claude --resume <id>` / `codex resume <id>`, loading that session's full history so you can pick up exactly where you left off. This was born from the built-in `/resume` picker rendering unreliably under a third-party provider.
>
> Licensed under the MIT License, with the original copyright notice retained.

![Interface preview](./assets/screenshots/example.png)

## Install

Install or update on macOS and Linux:

```bash
curl -LsSf https://raw.githubusercontent.com/zzusec/restore-session/main/install.sh | sh
```

Install or update on Windows from PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -c "irm https://raw.githubusercontent.com/zzusec/restore-session/main/install.ps1 | iex"
```

The installer installs `restore-session` and creates a short `re` alias pointing at it. On Windows it adds `%LOCALAPPDATA%\Programs\restore-session\bin` to your user `PATH`. Open a new terminal after the first install.

## Usage

```bash
restore-session
# The short alias `re` is also available (re codex, re claude, ...).
```

Run without arguments to choose an agent, or name one directly:

```bash
restore-session codex
restore-session claude
restore-session opencode
restore-session pi
```

By default, the app uses `$CODEX_HOME` or `~/.codex` for Codex, `$CLAUDE_CONFIG_DIR` or `~/.claude` for Claude Code, `$XDG_DATA_HOME/opencode` or `~/.local/share/opencode` for OpenCode, and `$PI_CODING_AGENT_DIR` or `~/.pi/agent` for Pi. On Windows, `~` is your user profile directory. To override a location explicitly:

```bash
restore-session --codex-home /path/to/codex
restore-session --claude-home /path/to/claude
restore-session --opencode-home /path/to/opencode
restore-session --pi-home /path/to/pi/agent
```

The OpenCode path must name the `opencode` data directory itself, not its parent. The `opencode.db` file, when present, lives directly inside it.

Session discovery and previews read the stored data directly. Codex changes and OpenCode deletions require their respective CLIs; if a CLI is unavailable, that agent opens in browse-only mode. Claude Code and Pi deletions operate directly on their session files.

### Language

The interface follows your locale (`LC_ALL`, `LC_MESSAGES`, `LANGUAGE`, or `LANG`). Chinese locales use Simplified Chinese; all other locales use English. To override detection:

```bash
RESTORE_SESSION_LANG=en restore-session
RESTORE_SESSION_LANG=zh-CN restore-session
```

In PowerShell, use `$env:RESTORE_SESSION_LANG = "zh-CN"` before running `restore-session`.

## Keyboard shortcuts

The footer shows the shortcuts available for the current agent and installation. Press `h` for the complete in-app reference.

| Key | Action | Supported by |
|---|---|---|
| `↑` `↓` / `j` `k` | Move to the previous / next session | All agents |
| `g` / `G` | Jump to the top / bottom of the list | All agents |
| `Tab` | Switch between the session list and conversation | All agents |
| `␣` | Select or deselect the current session | All agents |
| `/` | Search titles, working directories, session IDs, and clients | All agents |
| `?` | Search backward | All agents |
| `n` / `N` | Jump to the next / previous match | All agents |
| `c` | Copy the current session ID | All agents |
| `Enter` | Resume the current session (replaces this process with `claude --resume` / `codex resume`) | Claude Code, Codex |
| `y` | Copy the current session's working directory | All agents |
| `d` | Delete the current session or selected sessions | All agents |
| `a` | Archive the current session or selected sessions | Codex only |
| `u` | Unarchive the current session or selected sessions | Codex only |
| `D` | Delete all archived sessions | Codex only |
| `O` | Delete all orphaned sub-agent sessions | Codex and OpenCode |
| `E` | Delete all empty sessions | Claude Code only |
| `r` | Refresh the session list | All agents |
| `h` | Show keyboard shortcuts | All agents |
| `!` | Toggle danger mode; individual deletions skip confirmation | All agents |
| `Esc` | Exit multi-select, turn off danger mode, or clear the search | All agents |
| `q` | Quit | All agents |

Press Space or double-click to select or deselect a session. Selecting a session includes all of its descendant sub-agent sessions. Actions supported by the current agent apply to every selected session; press `Esc` to clear the selection.

## Resume a session (one key)

This is what `restore-session` is for. Move the cursor to any Claude Code or Codex session and press `Enter`: the current process is replaced by the agent resuming that session, with its full history loaded so you can continue right away.

```bash
restore-session claude      # browse Claude Code sessions, press Enter to resume
re codex           # same, for Codex (the `re` alias is identical)
```

| Agent | Key | What happens on Enter |
|---|---|---|
| Claude Code | `Enter` | `claude --resume <session-id>` takes over the terminal |
| Codex | `Enter` | `codex resume <session-id>` takes over the terminal |
| OpenCode / Pi | `c` then paste | resume via their CLI (Enter is browse-only here) |

If the agent binary is missing from your `PATH`, `restore-session` prints a message and exits instead of failing.

## Deletion and archiving

> [!WARNING]
> This tool does not create backups. Deleted sessions cannot be recovered.

- Deleting or archiving a session also includes its descendant sub-agent sessions.
- Codex archive, unarchive, and delete operations are delegated to the Codex CLI.
- Claude Code deletion removes the transcript and its related session data directly.
- OpenCode deletion is delegated to the OpenCode CLI.
- Pi deletion removes the session file directly.
- In danger mode (`!`), individual deletions skip confirmation. Bulk deletion still asks for confirmation.

## Acknowledgements

- [**agent-session-cleaner**](https://github.com/haowang02/agent-session-cleaner) by [Hao Wang (haowang02)](https://github.com/haowang02) — `restore-session` is a fork of this project. The session browsing, search, cleanup, and archiving features are entirely his work. Thank you.
- [LINUX DO](https://linux.do/) — a community for builders and curious minds (credited by the upstream project)
