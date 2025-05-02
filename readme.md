# nx

netcat + tmux - listen on a port and forward the stream to a tmux session

## Prerequisites

- tmux - required for session/window management
- socat - required for TTY handling and keystroke forwarding

## Usage

```
Flags:
-v - verbose
-l - listen
-p - port #
-t - target tmux session in which to create the windows. default: nx

--auto - autoupgrades the tty default: false

```

## Breakdown

1. start a listen, just like you would netcat
```sh
nx -vlp 9090 --auto -t myshells
```

2. from a different machine, initiate the connection
3. nx catches the connection, 
    - check for the existence of `myshells` or `nx` tmux session. creates if not exists
    - starts a tmux window in the session, and send the stream there
4. optionally, using tmux send-keys, upgrade the shell using `script` or python's `pty.spawn` functionality

## Testing

nx includes a test suite to ensure functionality works as expected:

### Running Tests

```sh
# Run all unit tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run integration tests (requires tmux)
RUN_INTEGRATION_TESTS=1 go test -tags integration ./...
```

### Test Structure

- **Unit Tests:** These test individual functions like file generation and socket handling
- **TTY Tests:** Tests for TTY functionality and keystroke handling
- **Integration Tests:** Tests full program flow with tmux

### Note on TTY Functionality

Raw mode must be enabled for proper keystroke forwarding to connections. The tests verify that keystrokes are properly passed from the TTY to the connection.
