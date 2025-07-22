# nx

[n]etcat + tmu[x] - listen on a port and forward the stream to a tmux session

## Prerequisites

- tmux - required for session/window management
- socat - required for TTY handling and keystroke forwarding

## Usage

1. start a listener
```sh
nx -vp 9090
```

2. from a different machine, initiate the reverse shell
3. nx catches the connection then starts a tmux window in the `nx` session, and starts the shell there

```
Usage:
  nx [OPTIONS]

Application Options:
      --auto     Attempt to auto-upgrade to a tty (deprecated: use --exec auto)
      --exec=    Execute plugin script on connection
  -i, --host=    Interface address on which to bind (default: 0.0.0.0)
  -p, --port=    Port on which to bind (default: 8443)
  -t, --target=  Tmux session name (default: nx)
  -v, --verbose  Debug logging
      --sleep=   adjust if --auto is failing (default: 500ms)
```

## Features

- **Plugin system**: Use `--exec <plugin>` to run custom scripts on connection
- **Auto-upgrade to TTY**: Use `--exec auto` (or deprecated `--auto` flag) to automatically upgrade your shell to a TTY
- **XDG runtime paths**: Automatically uses XDG runtime directory for socket location
- **Signal handling**: Properly handles signals and performs cleanup

## Plugin System

nx supports a plugin system for executing custom commands when a new connection is established. Plugins are shell scripts stored in `~/.config/nx/plugins/`.

### Creating Plugins

1. Create a shell script in `~/.config/nx/plugins/<name>.sh`
2. Make it executable: `chmod +x ~/.config/nx/plugins/<name>.sh`
3. Use the plugin: `nx --exec <name>`

### Plugin Format

Plugins are simple shell scripts with these features:
- Lines starting with `#` are ignored (comments)
- Empty lines are ignored
- All other lines are executed as tmux commands in the reverse shell

### Example Plugin

```bash
#!/bin/bash
# Custom upgrade script
echo "Setting up environment..."
export PATH=/usr/local/bin:$PATH
whoami
```

### Built-in Plugins

- **auto**: TTY upgrade script (equivalent to deprecated `--auto` flag)

## How?

unix domain sockets mannn

## ToDo
- [x] ~~maybe a plugin system for sending commands on connection~~ ✅ **DONE**: Implemented plugin system with `--exec` flag
- [x] some mechanism to auto-upgrade the shell to a TTY via tmux-send-keys or sourcing a script that just adds the keybinds, so that it's up to the user to fire off the upgrade
- [ ] alternatively, multiplex the connection to allow `curl | sh` from the same port
- [ ] handle stdio with the socket directly with `nx`, eliminating the need for `socat`
- [ ] facilitate installing plugins dir to xdg
- [ ] multiplexing listener
  - [ ] same port, but detects plain revshell vs http request. could allow scripts to be `curl |sh`d instead of relying on `tmux send-keys`
  - [ ] super simple chisel-light
