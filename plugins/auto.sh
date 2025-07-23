#!/bin/bash

# Auto-upgrade plugin for nx
# Attempts to upgrade a reverse shell to a proper TTY

# Try to spawn an interactive bash shell using expect
script -qc /bin/bash /dev/null
# expect -c 'spawn bash; interact'
# python3 -c 'import pty; pty.spawn("/bin/bash")'

# Background the process with Ctrl+Z
C-z

# Fix terminal settings for raw mode
stty size; stty raw -echo; fg

# Set proper terminal type for colors
export TERM=xterm-256color
