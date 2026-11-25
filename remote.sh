# Server Firefox remote script. Runs on the target server as root, fed via
# stdin by the panel with SF_ACTION, SF_DOMAIN, SF_USER, SF_PASS, SF_RES set.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
step() { printf '==> %s\n' "$*"; }
die() { printf 'ERROR: %s\n' "$*"; exit 1; }
[ "$(id -u)" -eq 0 ] || die "root (or passwordless sudo) is required"
UNITS="sf-xvfb sf-wm sf-firefox sf-x11vnc sf-websockify"
B=sfbrowser
H=/home/$B

status() {
  for u in $UNITS nginx; do printf '%-16s %s\n' "$u" "$(systemctl is-active $u 2>/dev/null || true)"; done
  if [ -n "${SF_DOMAIN:-}" ] && [ -f /etc/letsencrypt/live/$SF_DOMAIN/fullchain.pem ]; then
    printf 'SSL until         %s\n' "$(openssl x509 -enddate -noout -in /etc/letsencrypt/live/$SF_DOMAIN/fullchain.pem | cut -d= -f2)"
  fi
  free -h | sed -n 2p | awk '{print "RAM               "$3" / "$2}'
  df -h / | awk 'NR==2{print "Disk              "$3" / "$2}'
}

case "$SF_ACTION" in
status) status; exit 0 ;;
restart)
  step "Restarting Firefox and the display"
  systemctl restart $UNITS; sleep 3; status
  for u in $UNITS; do systemctl is-active --quiet $u || die "$u is not running (journalctl -u $u)"; done
  exit 0 ;;
uninstall)
  step "Removing Server Firefox from this server"
  systemctl disable --now $UNITS 2>/dev/null || true
  for u in $UNITS; do rm -f /etc/systemd/system/$u.service; done
  systemctl daemon-reload
  rm -f /etc/nginx/sites-enabled/server-firefox /etc/nginx/sites-available/server-firefox /etc/nginx/.sf-htpasswd
  systemctl reload nginx 2>/dev/null || true
  userdel -r $B 2>/dev/null || true
  echo "Firefox package, nginx and certificates were kept."
  exit 0 ;;
install) ;;
*) die "unknown action $SF_ACTION" ;;
esac

# ---------------------------------------------------------------- install
echo "$SF_DOMAIN" | grep -Eq '^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[A-Za-z]{2,}$' || die "invalid domain: $SF_DOMAIN"
[ ${#SF_PASS} -ge 8 ] || die "Firefox login password must be at least 8 characters"
echo "$SF_RES" | grep -Eq '^[0-9]{3,4}x[0-9]{3,4}$' || SF_RES=1920x1080
. /etc/os-release
case "$ID:$VERSION_ID" in
  ubuntu:22.04|ubuntu:24.04|debian:12) echo "OS: $PRETTY_NAME" ;;
  *) die "supported: Ubuntu 22.04, Ubuntu 24.04, Debian 12 (this is ${PRETTY_NAME:-unknown})" ;;
esac

step "Checking that $SF_DOMAIN points to this server"
MYIP=$(curl -4 -fsS --max-time 8 https://api.ipify.org || true)
DIP=$(getent ahostsv4 "$SF_DOMAIN" | awk 'NR==1{print $1}' || true)
echo "server IP: ${MYIP:-?}   domain IP: ${DIP:-not resolved}"
[ -n "$DIP" ] || die "$SF_DOMAIN does not resolve yet; set its A record to ${MYIP:-the IP of this server}"
[ -z "$MYIP" ] || [ "$MYIP" = "$DIP" ] || echo "WARNING: domain IP differs from server IP (fine only behind a proxy/NAT)"

step "Installing packages"
if [ "$ID" = ubuntu ] && ! grep -rhqs '^deb .* universe' /etc/apt/sources.list /etc/apt/sources.list.d/ && ! grep -rqs 'universe' /etc/apt/sources.list.d/*.sources; then
  apt-get update -q && apt-get install -yq software-properties-common && add-apt-repository -y universe
fi
apt-get update -q
apt-get install -yq ca-certificates curl gnupg wget openssl xvfb x11vnc xauth x11-utils \
  matchbox-window-manager novnc websockify nginx apache2-utils certbot \
  dbus-x11 fonts-liberation fonts-noto-color-emoji fonts-noto-core 2>&1 | grep -E '^(Setting up|E:)' || true
for c in Xvfb x11vnc websockify nginx certbot htpasswd matchbox-window-manager; do command -v $c >/dev/null || die "$c failed to install"; done

step "Installing Firefox from Mozilla's official repository"
install -d -m 0755 /etc/apt/keyrings
wget -q https://packages.mozilla.org/apt/repo-signing-key.gpg -O /etc/apt/keyrings/packages.mozilla.org.asc
echo 'deb [signed-by=/etc/apt/keyrings/packages.mozilla.org.asc] https://packages.mozilla.org/apt mozilla main' >/etc/apt/sources.list.d/mozilla.list
printf 'Package: *\nPin: origin packages.mozilla.org\nPin-Priority: 1000\n' >/etc/apt/preferences.d/mozilla
if snap list firefox >/dev/null 2>&1; then snap remove firefox || true; fi
apt-get update -q
apt-get install -yq --allow-downgrades firefox 2>&1 | grep -E '^(Setting up|E:)' || true
command -v firefox >/dev/null || die "Firefox failed to install"
firefox --version 2>/dev/null || true

step "Creating the browser user and profile"
id $B >/dev/null 2>&1 || useradd --create-home --shell /usr/sbin/nologin $B
PROF=$H/.mozilla/profile
install -d -m 700 -o $B -g $B $H/.mozilla $PROF $H/.cache
cat >$PROF/user.js <<'JS'
user_pref("browser.shell.checkDefaultBrowser", false);
user_pref("browser.aboutwelcome.enabled", false);
user_pref("datareporting.policy.dataSubmissionPolicyBypassNotification", true);
user_pref("browser.startup.homepage_override.mstone", "ignore");
user_pref("browser.sessionstore.resume_from_crash", true);
user_pref("browser.tabs.warnOnClose", false);
user_pref("gfx.webrender.software", true);
JS
chown -R $B:$B $H
chmod 700 $H

step "Writing services (display $SF_RES, full-screen window manager)"
unit() { cat >/etc/systemd/system/$1.service; }
unit sf-xvfb <<U
[Unit]
Description=Server Firefox: virtual display
After=network.target
[Service]
User=$B
ExecStart=/usr/bin/Xvfb :1 -screen 0 ${SF_RES}x24 -nolisten tcp -dpi 96
Restart=always
RestartSec=2
[Install]
WantedBy=multi-user.target
U
WAIT='/usr/bin/timeout 30 /bin/sh -c "until xdpyinfo -display :1 >/dev/null 2>&1; do sleep 1; done"'
unit sf-wm <<U
[Unit]
Description=Server Firefox: window manager (keeps Firefox full screen)
Requires=sf-xvfb.service
After=sf-xvfb.service
[Service]
User=$B
Environment=DISPLAY=:1
ExecStartPre=$WAIT
ExecStart=/usr/bin/matchbox-window-manager -use_titlebar no -use_cursor yes
Restart=always
RestartSec=2
[Install]
WantedBy=multi-user.target
U
unit sf-firefox <<U
[Unit]
Description=Server Firefox: browser
Requires=sf-xvfb.service
After=sf-wm.service
Wants=sf-wm.service
[Service]
User=$B
Environment=DISPLAY=:1 HOME=$H MOZ_ENABLE_WAYLAND=0
WorkingDirectory=$H
ExecStartPre=$WAIT
ExecStart=/usr/bin/firefox --no-remote --profile $PROF
Restart=always
RestartSec=3
[Install]
WantedBy=multi-user.target
U
unit sf-x11vnc <<U
[Unit]
Description=Server Firefox: VNC (localhost only)
Requires=sf-xvfb.service
After=sf-xvfb.service
[Service]
User=$B
ExecStartPre=$WAIT
ExecStart=/usr/bin/x11vnc -display :1 -forever -shared -localhost -rfbport 5901 -nopw -noxdamage -quiet
Restart=always
RestartSec=2
[Install]
WantedBy=multi-user.target
U
unit sf-websockify <<U
[Unit]
Description=Server Firefox: noVNC bridge (localhost only)
Requires=sf-x11vnc.service
After=sf-x11vnc.service
[Service]
User=$B
ExecStart=/usr/bin/websockify --web=/usr/share/novnc/ 127.0.0.1:6081 127.0.0.1:5901
Restart=always
RestartSec=2
[Install]
WantedBy=multi-user.target
U
systemctl daemon-reload
systemctl enable $UNITS >/dev/null 2>&1

step "Nginx (port 80, login required from the start)"
printf '%s\n' "$SF_PASS" | htpasswd -iB -c /etc/nginx/.sf-htpasswd "$SF_USER" >/dev/null 2>&1
chown root:www-data /etc/nginx/.sf-htpasswd; chmod 640 /etc/nginx/.sf-htpasswd
mkdir -p /var/www/sf-acme
PROXY='
    auth_basic "Firefox";
    auth_basic_user_file /etc/nginx/.sf-htpasswd;
    location = / { return 302 /vnc.html?autoconnect=1&resize=scale&reconnect=1&path=websockify; }
    location /websockify {
        proxy_pass http://127.0.0.1:6081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_read_timeout 86400;
        proxy_buffering off;
    }
    location / { proxy_pass http://127.0.0.1:6081; proxy_buffering off; }'
cat >/etc/nginx/sites-available/server-firefox <<N
server {
    listen 80;
    listen [::]:80;
    server_name $SF_DOMAIN;
    location /.well-known/acme-challenge/ { root /var/www/sf-acme; auth_basic off; }
$PROXY
}
N
ln -sf /etc/nginx/sites-available/server-firefox /etc/nginx/sites-enabled/server-firefox
[ -L /etc/nginx/sites-enabled/default ] && rm -f /etc/nginx/sites-enabled/default
nginx -t 2>&1
systemctl enable nginx >/dev/null 2>&1; systemctl reload-or-restart nginx
if command -v ufw >/dev/null && ufw status | grep -q 'Status: active'; then ufw allow 80/tcp >/dev/null; ufw allow 443/tcp >/dev/null; echo "ufw: 80 and 443 opened"; fi

step "Getting the SSL certificate for $SF_DOMAIN"
certbot certonly --webroot -w /var/www/sf-acme -d "$SF_DOMAIN" -n --agree-tos \
  --register-unsafely-without-email --keep-until-expiring \
  --deploy-hook "systemctl reload nginx" 2>&1 || die "certificate failed: make sure $SF_DOMAIN points here and port 80 is open"
CRT=/etc/letsencrypt/live/$SF_DOMAIN
cat >/etc/nginx/sites-available/server-firefox <<N
server {
    listen 80;
    listen [::]:80;
    server_name $SF_DOMAIN;
    location /.well-known/acme-challenge/ { root /var/www/sf-acme; }
    location / { return 301 https://\$host\$request_uri; }
}
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name $SF_DOMAIN;
    ssl_certificate $CRT/fullchain.pem;
    ssl_certificate_key $CRT/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    add_header Strict-Transport-Security "max-age=31536000" always;
$PROXY
}
N
nginx -t >/dev/null 2>&1 || sed -i 's/^    http2 on;$//' /etc/nginx/sites-available/server-firefox
nginx -t 2>&1
systemctl reload nginx
systemctl enable --now certbot.timer >/dev/null 2>&1 || true

step "Starting Firefox"
systemctl restart $UNITS
sleep 5
status
for u in $UNITS nginx; do systemctl is-active --quiet $u || { journalctl -u $u -n 15 --no-pager; die "$u is not running"; }; done
for p in 5901 6081; do
  ss -lnt | awk '{print $4}' | grep -Eq "^(0\.0\.0\.0|\[::\]|\*):${p}\$" && die "port $p is public; it must be localhost only"
done
code=$(curl -s -o /dev/null -w '%{http_code}' --resolve "$SF_DOMAIN:443:127.0.0.1" "https://$SF_DOMAIN/vnc.html" || true)
[ "$code" = 401 ] || die "HTTPS check returned $code (expected 401 login prompt)"
echo "Open: https://$SF_DOMAIN"
