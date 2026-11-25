#!/usr/bin/env bash
# Server Firefox installer:
#   bash <(curl -fsSL https://github.com/sahandfarshad/server-firefox/releases/latest/download/install.sh)
# Installs the panel on this (main) server, asks for a port, creates a secret
# panel path and password, then opens the SSH menu (`sf`).
set -euo pipefail
REPO="sahandfarshad/server-firefox"
BIN=/usr/local/bin/server-firefox
export SF_DIR="${SF_DIR:-/var/lib/server-firefox}"
UNIT=/etc/systemd/system/server-firefox.service
gold=$'\033[38;5;208m'; dim=$'\033[2m'; red=$'\033[31m'; rst=$'\033[0m'
say() { printf '  %s\n' "$*"; }
die() { printf '  %sError:%s %s\n' "$red" "$rst" "$*" >&2; exit 1; }
[ "$(id -u)" -eq 0 ] || die "run as root (sudo -i)"
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;;
  *) die "unsupported architecture: $(uname -m)" ;;
esac
command -v curl >/dev/null || die "curl is missing (apt install curl)"
TTY=""
if [ -t 0 ]; then TTY=/dev/stdin; elif [ -r /dev/tty ] && (exec </dev/tty) 2>/dev/null; then TTY=/dev/tty; fi
ask() { REPLY=""; if [ -n "$TTY" ]; then read -r -p "  $1 [$2]: " REPLY <"$TTY" || true; fi; REPLY="${REPLY:-$2}"; }

echo; printf '  %s✦ Server Firefox%s  %s\n\n' "$gold" "$rst" "${dim}installer · linux-$ARCH${rst}"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
GH="https://github.com/$REPO/releases/latest/download"
say "Downloading the latest release"
curl -fsSL --retry 2 -o "$TMP/SHA256SUMS" "$GH/SHA256SUMS" || die "GitHub is not reachable"
curl -fL --retry 2 --progress-bar -o "$TMP/server-firefox-linux-$ARCH" "$GH/server-firefox-linux-$ARCH"
(cd "$TMP" && grep " server-firefox-linux-$ARCH\$" SHA256SUMS | sha256sum -c --quiet -) || die "checksum mismatch"
systemctl is-active --quiet server-firefox 2>/dev/null && systemctl stop server-firefox
install -m 755 "$TMP/server-firefox-linux-$ARCH" "$BIN"
ln -sf "$BIN" /usr/local/bin/sf
say "Installed $("$BIN" version)  (menu: sf)"

if [ ! -f "$SF_DIR/config.json" ]; then
  SUG="$("$BIN" freeport)"
  while :; do
    ask "Port for the panel (a free one is suggested)" "$SUG"; P="$REPLY"
    if ! [[ "$P" =~ ^[0-9]+$ ]] || [ "$P" -lt 1 ] || [ "$P" -gt 65535 ]; then say "${red}Not a valid port.${rst}"
    elif ! "$BIN" portfree "$P"; then say "${red}Port $P is already in use.${rst}"
    else break; fi
    [ -n "$TTY" ] || die "port $P is not usable"
    SUG="$("$BIN" freeport)"
  done
  "$BIN" init --port "$P" >/dev/null
else
  say "Existing installation found: settings and servers kept."
fi
PORT="$(sed -n 's/.*"port": *\([0-9]*\).*/\1/p' "$SF_DIR/config.json" | head -n1)"

cat >"$UNIT" <<UNITEOF
[Unit]
Description=Server Firefox (panel)
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=$BIN serve
Environment=SF_DIR=$SF_DIR
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
UNITEOF
systemctl daemon-reload
systemctl enable server-firefox >/dev/null 2>&1
systemctl restart server-firefox
sleep 1
systemctl is-active --quiet server-firefox || say "${red}The service did not start: journalctl -u server-firefox${rst}"
if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow "$PORT/tcp" >/dev/null && say "Firewall (ufw): port $PORT opened."
fi
"$BIN" info
if [ -n "$TTY" ] && [ "${SF_NO_MENU:-}" != 1 ]; then "$BIN" menu <"$TTY" || true; fi
