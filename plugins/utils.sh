
function linpeas() {
		url="https://github.com/peass-ng/PEASS-ng/releases/latest/download/linpeas.sh";
		(curl -sL "${url}" || wget -O - "${url}") | sh
}

function chizl() {
		IFS=: read HOST PORT <<< $ME;
		ssh -fNT "$@" -p "${PORT}" "${HOST}"
}
