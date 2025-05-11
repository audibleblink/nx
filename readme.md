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
      --auto     Attempt to auto-upgrade to a tty with customizable sleep time
  -i, --host=    Interface address on which to bind (default: 0.0.0.0)
  -p, --port=    Port on which to bind (default: 8443)
  -t, --target=  Tmux session name (default: nx)
  -v, --verbose  Debug logging
  --sleep=   adjust if --auto is failing (default: 500ms)
```

## Features

- **Auto-upgrade to TTY**: Use `--auto` flag to automatically upgrade your shell to a TTY
- **XDG runtime paths**: Automatically uses XDG runtime directory for socket location
- **Signal handling**: Properly handles signals and performs cleanup

## How?

unix domain sockets mannn

## ToDo
- [ ] maybe a plugin system for sending commands on connection
- [ ] alternatively, multiplex the connection to allow `curl | sh` from the same port
