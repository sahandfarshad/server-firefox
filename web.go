package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

//go:embed ui.html
var uiHTML []byte

type Panel struct {
	st  *Store
	run *Runner
}

func NewPanel(st *Store) (*Panel, error) { return &Panel{st: st, run: NewRunner(st)}, nil }

func orStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

var (
	domainRe = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[A-Za-z]{2,}$`)
	userRe   = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,32}$`)
	resRe    = regexp.MustCompile(`^[0-9]{3,4}x[0-9]{3,4}$`)
)

func (p *Panel) Routes() http.Handler {
	base := p.st.Cfg.AdminPath
	mux := http.NewServeMux()
	mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != base {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(uiHTML)
	})
	mux.HandleFunc(base+"api/login", p.login)
	mux.HandleFunc(base+"api/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sf", Value: "", Path: base, MaxAge: -1})
		w.WriteHeader(204)
	})
	mux.HandleFunc(base+"api/servers", p.auth(p.servers))
	mux.HandleFunc(base+"api/delete", p.auth(p.deleteServer))
	mux.HandleFunc(base+"api/run", p.auth(p.runAction))
	mux.HandleFunc(base+"api/log", p.auth(p.logs))
	return secure(mux)
}

func secure(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method == http.MethodPost && r.Header.Get("X-SF") != "1" { // CSRF guard: only our own fetch() sets it
			http.Error(w, "forbidden", 403)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func jsonOut(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func errOut(w http.ResponseWriter, code int, msg string) {
	jsonOut(w, code, map[string]string{"error": msg})
}

func (p *Panel) sign(exp int64) string {
	m := hmac.New(sha256.New, []byte(p.st.Cfg.Secret+p.st.Cfg.PassHash))
	m.Write([]byte(strconv.FormatInt(exp, 10)))
	return strconv.FormatInt(exp, 10) + "." + hex.EncodeToString(m.Sum(nil))
}

func (p *Panel) authed(r *http.Request) bool {
	c, err := r.Cookie("sf")
	if err != nil {
		return false
	}
	i := strings.IndexByte(c.Value, '.')
	if i < 0 {
		return false
	}
	exp, err := strconv.ParseInt(c.Value[:i], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(p.sign(exp)), []byte(c.Value)) == 1
}

func (p *Panel) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !p.authed(r) {
			errOut(w, 401, "login required")
			return
		}
		h(w, r)
	}
}

func (p *Panel) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonOut(w, 200, map[string]any{"authed": p.authed(r), "version": version})
		return
	}
	var in struct{ User, Pass string }
	json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in)
	okUser := subtle.ConstantTimeCompare([]byte(in.User), []byte(p.st.Cfg.User)) == 1
	if bcrypt.CompareHashAndPassword([]byte(p.st.Cfg.PassHash), []byte(in.Pass)) != nil || !okUser {
		time.Sleep(1200 * time.Millisecond)
		errOut(w, 401, "wrong username or password")
		return
	}
	exp := time.Now().Add(7 * 24 * time.Hour).Unix()
	http.SetCookie(w, &http.Cookie{Name: "sf", Value: p.sign(exp), Path: p.st.Cfg.AdminPath, HttpOnly: true,
		SameSite: http.SameSiteStrictMode, Secure: p.st.Cfg.TLS(), Expires: time.Unix(exp, 0)})
	jsonOut(w, 200, map[string]any{"authed": true})
}

type serverView struct {
	ID, Name, Host, SSHUser, Domain, WebUser, Resolution, HostKey, Status, LastError, Updated, URL, Running string
	SSHPort                                                                                                 int
}

func (p *Panel) servers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		p.saveServer(w, r)
		return
	}
	list, err := p.st.Servers()
	if err != nil {
		errOut(w, 500, err.Error())
		return
	}
	out := []serverView{}
	for _, s := range list {
		v := serverView{ID: s.ID, Name: s.Name, Host: s.Host, SSHPort: s.SSHPort, SSHUser: s.SSHUser, Domain: s.Domain,
			WebUser: s.WebUser, Resolution: s.Resolution, HostKey: s.HostKey, Status: s.Status, LastError: s.LastError,
			Updated: s.Updated, URL: "https://" + s.Domain}
		if j := p.run.Job(s.ID); j != nil {
			j.mu.Lock()
			if !j.Done {
				v.Running = j.Action
			}
			j.mu.Unlock()
		}
		out = append(out, v)
	}
	jsonOut(w, 200, out)
}

func (p *Panel) saveServer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID, Name, Host, SSHUser, SSHPass, Domain, WebUser, WebPass, Resolution string
		SSHPort                                                                int
		ClearHostKey                                                           bool
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil {
		errOut(w, 400, "bad request")
		return
	}
	sv := &Server{ID: randHex(10), Status: "new"}
	if in.ID != "" {
		if sv = p.st.Server(in.ID); sv == nil {
			errOut(w, 404, "server not found")
			return
		}
	}
	in.Host = strings.Trim(strings.TrimSpace(in.Host), "[]")
	in.Domain = strings.ToLower(strings.TrimSpace(in.Domain))
	switch {
	case in.Host == "" || (net.ParseIP(in.Host) == nil && !domainRe.MatchString(in.Host)):
		errOut(w, 400, "invalid server IP or host")
		return
	case !domainRe.MatchString(in.Domain):
		errOut(w, 400, "invalid domain (example: fire.example.com)")
		return
	case !userRe.MatchString(in.WebUser):
		errOut(w, 400, "Firefox username: 3 to 32 letters, digits, dot, dash or underscore")
		return
	case in.WebPass == "" && sv.WebPass == "", in.WebPass != "" && len(in.WebPass) < 8:
		errOut(w, 400, "Firefox password must be at least 8 characters")
		return
	case in.SSHPass == "" && sv.SSHPass == "":
		errOut(w, 400, "SSH password is required")
		return
	}
	if in.SSHPort < 1 || in.SSHPort > 65535 {
		in.SSHPort = 22
	}
	if !resRe.MatchString(in.Resolution) {
		in.Resolution = "1920x1080"
	}
	sv.Name = orStr(strings.TrimSpace(in.Name), in.Domain)
	if sv.Host != in.Host || sv.SSHPort != in.SSHPort || in.ClearHostKey {
		sv.HostKey = ""
	}
	sv.Host, sv.SSHPort, sv.SSHUser = in.Host, in.SSHPort, orStr(strings.TrimSpace(in.SSHUser), "root")
	sv.Domain, sv.WebUser, sv.Resolution = in.Domain, in.WebUser, in.Resolution
	if in.SSHPass != "" {
		sv.SSHPass = p.st.Encrypt(in.SSHPass)
	}
	if in.WebPass != "" {
		sv.WebPass = p.st.Encrypt(in.WebPass)
	}
	if err := p.st.SaveServer(sv); err != nil {
		errOut(w, 500, err.Error())
		return
	}
	jsonOut(w, 200, map[string]string{"id": sv.ID})
}

func (p *Panel) deleteServer(w http.ResponseWriter, r *http.Request) {
	var in struct{ ID string }
	json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in)
	if j := p.run.Job(in.ID); j != nil && !j.Done {
		errOut(w, 409, "an action is still running on this server")
		return
	}
	p.st.UpdateServers(func(l []*Server) []*Server {
		out := l[:0]
		for _, x := range l {
			if x.ID != in.ID {
				out = append(out, x)
			}
		}
		return out
	})
	w.WriteHeader(204)
}

func (p *Panel) runAction(w http.ResponseWriter, r *http.Request) {
	var in struct{ ID, Action string }
	json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in)
	switch in.Action {
	case "install", "status", "restart", "uninstall":
	default:
		errOut(w, 400, "unknown action")
		return
	}
	if err := p.run.Start(in.ID, in.Action); err != nil {
		errOut(w, 409, err.Error())
		return
	}
	w.WriteHeader(204)
}

func (p *Panel) logs(w http.ResponseWriter, r *http.Request) {
	j := p.run.Job(r.URL.Query().Get("id"))
	if j == nil {
		jsonOut(w, 200, map[string]any{"lines": []string{}, "next": 0, "done": true, "none": true})
		return
	}
	from, _ := strconv.Atoi(r.URL.Query().Get("from"))
	lines, next, done, ok := j.snapshot(from)
	jsonOut(w, 200, map[string]any{"lines": lines, "next": next, "done": done, "ok": ok, "action": j.Action})
}
