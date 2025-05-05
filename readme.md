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

## How?

unix domain sockets mannn

## ToDo
- [ ] recover ctrl-c whoopsies since the reverse shell is tied to a unix domain socket and not a terminal
- [ ] some mechanism to auto-upgrade the shell to a TTY via tmux-send-keys or sourcing a script that just adds the keybinds, so that it's up to the user to fire off the upgrade
