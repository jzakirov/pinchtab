# CLI Overview

`pinchtab` has two normal usage styles:

- interactive menu mode
- direct command mode

Use menu mode when you want a guided local control surface.
Use direct commands when you want a faster shell workflow or want to script PinchTab.

## Interactive Menu

When you run `pinchtab` with no subcommand in an interactive terminal, it shows the startup banner and main menu.

Typical flow:

```text
listen    running  127.0.0.1:9867
str,plc   simple,fcfs
daemon    ok
security  [■■■■■■■■■■]  LOCKED

Main Menu
  1. Start server
  2. Daemon
  3. Start bridge
  4. Start MCP server
  5. Config
  6. Security
  7. Help
  8. Exit
```

What each entry does:

- `Start server` starts the full PinchTab server
- `Daemon` shows background service status and actions
- `Start bridge` starts the single-instance bridge runtime
- `Start MCP server` starts the stdio MCP server
- `Config` opens the interactive config screen
- `Security` opens the interactive security screen
- `Help` shows the command help tree

## Direct Commands

You can always bypass the menu and call commands directly.

Common examples:

```bash
pinchtab server
pinchtab daemon
pinchtab config
pinchtab security
pinchtab nav https://example.com
pinchtab snap -i -c
pinchtab click e5
pinchtab text
```

Direct commands are the better fit when:

- you are scripting PinchtTab
- you want repeatable shell history
- you are calling PinchtTab from another tool
- you already know which command you want

## Core Local Commands

These are the main local-control commands surfaced in the menu:

| Command | Purpose |
| --- | --- |
| `pinchtab` | Open the interactive menu in a terminal, or start the server in non-interactive use |
| `pinchtab server` | Start the full server and dashboard |
| `pinchtab daemon` | Show daemon status and manage the background service |
| `pinchtab config` | Open the interactive config overview/editor |
| `pinchtab security` | Review or change the current security posture |
| `pinchtab completion <shell>` | Generate shell completion scripts for `bash`, `zsh`, `fish`, or `powershell` |
| `pinchtab bridge` | Start the single-instance bridge runtime |
| `pinchtab mcp` | Start the stdio MCP server |

## Shell Completion

Use the built-in completion command to generate shell-specific scripts:

```bash
# Generate and install zsh completions
pinchtab completion zsh > "${fpath[1]}/_pinchtab"

# Generate bash completions
pinchtab completion bash > /etc/bash_completion.d/pinchtab

# Generate fish completions
pinchtab completion fish > ~/.config/fish/completions/pinchtab.fish
```

## Browser Shortcuts

The most common browser control shortcuts are top-level commands:

| Command | Purpose |
| --- | --- |
| `pinchtab nav <url>` | Navigate to a URL |
| `pinchtab quick <url>` | Navigate and analyze the page |
| `pinchtab snap` | Get an accessibility snapshot |
| `pinchtab click <ref>` | Click an element ref |
| `pinchtab type <ref> <text>` | Type into an element |
| `pinchtab fill <ref|selector> <text>` | Fill an input directly |
| `pinchtab text` | Extract page text |
| `pinchtab screenshot` | Capture a screenshot |
| `pinchtab pdf` | Export the current page as PDF |
| `pinchtab health` | Check server health |

## Config From The CLI

`pinchtab config` now acts as the main interactive config screen.

It shows:

- instance strategy
- allocation policy
- default stealth level
- default tab eviction policy
- config file path
- dashboard URL when the server is running
- the masked server token
- a `Copy token` action for clipboard/manual copy

For exact config commands and schema details, see [Config](./config.md).

## Security From The CLI

`pinchtab security` is the main interactive security screen.

Use it to:

- review the current posture
- inspect warnings
- edit individual security controls
- apply `security up`
- apply `security down`

The direct subcommands also exist:

```bash
pinchtab security up
pinchtab security down
```

For broader security guidance, see [Security Guide](../guides/security.md).

## Daemon From The CLI

`pinchtab daemon` shows status, recent logs, and available actions.

The command is supported on:

- macOS via `launchd`
- Linux via user `systemd`

It will fail fast when the current environment cannot manage a user service, for example:

- Linux shells without a working `systemctl --user` session
- macOS sessions without an active GUI `launchd` domain

For operational details, see [Background Service (Daemon)](../guides/daemon.md).

## Full Command Tree

Use the built-in help for the current command tree:

```bash
pinchtab --help
```

For per-command reference pages, start at [Reference Index](./index.md).
