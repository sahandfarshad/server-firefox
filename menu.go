package main

// SSH (terminal) menu: `server-firefox`, `server-firefox menu` or `sp`.

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	cReset = "\033[0m"
	cBold  = "\033[1m"
	cDim   = "\033[2m"
	cGold  = "\033[38;5;208m"
	cGreen = "\033[32m"
	cRed   = "\033[31m"
	cYel   = "\033[33m"
)

func bold(s string) string { return cBold + s + cReset }
func dim(s string) string  { return cDim + s + cReset }
func gold(s string) string { return cGold + s + cReset }
func clearScreen()         { fmt.Print("\033[H\033[2J") }

var stdin = bufio.NewReader(os.Stdin)

func prompt(q, def string) string {
	if def != "" {
		fmt.Printf("  %s [%s]: ", q, def)
	} else {
		fmt.Printf("  %s: ", q)
	}
	s, _ := stdin.ReadString('\n')
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	return s
}

func yes(q string) bool {
	a := strings.ToLower(prompt(q+" (y/N)", ""))
	return a == "y" || a == "yes"
}

func readSecret(q string) string {
	fmt.Print("  " + q + ": ")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, _ := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		return string(b)
	}
	s, _ := stdin.ReadString('\n')
	return strings.TrimSpace(s)
}

func plainPassPath(dir string) string { return filepath.Join(dir, "admin-password.txt") }

// The generated password is kept (root-only) so the menu can show it until
// the owner picks their own.
func writePlainPass(dir, pass string) {
	if pass == "" {
		os.Remove(plainPassPath(dir))
		return
	}
	_ = os.WriteFile(plainPassPath(dir), []byte(pass+"\n"), 0o600)
}

func readPlainPass(dir string) string {
	b, _ := os.ReadFile(plainPassPath(dir))
	return strings.TrimSpace(string(b))
}

func hasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func svc(args ...string) string {
	if !hasSystemd() {
		return ""
	}
	out, _ := exec.Command("systemctl", args...).CombinedOutput()
	return strings.TrimSpace(string(out))
}

func restartService() {
	if hasSystemd() {
		svc("restart", svcName)
		time.Sleep(700 * time.Millisecond)
		fmt.Println("  Service restarted: " + svc("is-active", svcName))
	}
}

func dot(ok bool, label string) string {
	if ok {
		return cGreen + "● " + label + cReset
	}
	return cRed + "● " + label + cReset
}

func drawMenu(st *Store) {
	clearScreen()
	c := st.Cfg
	state := svc("is-active", svcName)
	const w = 51
	title, ver := "✦ Server Firefox", "v"+version
	pad := w - 2 - len([]rune(title)) - len(ver)
	fmt.Println()
	fmt.Println("  " + dim("╭"+strings.Repeat("─", w)+"╮"))
	fmt.Println("  " + dim("│") + " " + cGold + cBold + title + cReset + strings.Repeat(" ", max(pad, 1)) + dim(ver) + " " + dim("│"))
	fmt.Println("  " + dim("╰"+strings.Repeat("─", w)+"╯"))
	row := func(k, v string) { fmt.Printf("   %s%-7s%s %s\n", cDim, k, cReset, v) }
	row("Panel", bold(siteURL(st))+"  "+dot(state == "active", orStr(state, "no systemd")))
	row("Admin", siteURL(st)+c.AdminPath)
	row("User", c.User)
	ssl := cYel + "off" + cReset
	if c.TLS() {
		ssl = c.Domain
		if exp, err := certExpiry(c.CertFile); err == nil {
			ssl += dim("  until " + exp.Format("2006-01-02"))
		}
	}
	row("SSL", ssl)
	section := func(name string) {
		fmt.Println()
		fmt.Println("   " + cDim + cBold + name + cReset + " " + dim(strings.Repeat("─", w-len(name)-1)))
	}
	item := func(n, label string) { fmt.Printf("   %s%3s%s  %s %s\n", cGold+cBold, n, cReset, dim("│"), label) }
	section("PANEL")
	item("1", "Status")
	item("2", "Change port")
	item("3", "Domain + SSL (Let's Encrypt)")
	item("4", "Remove SSL (back to HTTP)")
	item("5", "Admin password (show / reset)")
	item("6", "Admin path")
	section("SYSTEM")
	item("7", "Restart")
	item("8", "Update")
	item("9", cRed+"Uninstall"+cReset)
	fmt.Println()
	item("0", dim("Exit"))
	fmt.Println()
}

func runMenu(dir string) {
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		initDefaults(dir, 0)
	}
	for {
		st := mustStore(dir)
		drawMenu(st)
		choice := prompt(gold("›")+" Choose", "")
		fmt.Println()
		switch choice {
		case "1":
			showStatus(st)
		case "2":
			sug := suggestPort()
			p, err := strconv.Atoi(prompt(fmt.Sprintf("New port (current %d, a free one suggested)", st.Cfg.Port), strconv.Itoa(sug)))
			if err != nil || p < 1 || p > 65535 {
				fmt.Println("  Invalid port.")
				break
			}
			if p != st.Cfg.Port && !portFree(p) {
				fmt.Printf("  Port %d is already in use. Nothing changed.\n", p)
				break
			}
			st.Cfg.Port = p
			saveAndRestart(st)
			fmt.Println("  New address: " + bold(siteURL(st)))
			fmt.Println(dim("  Remember to open the port in your firewall if you use one."))
		case "3":
			sslSetup(st)
		case "4":
			if !st.Cfg.TLS() {
				fmt.Println("  SSL is already off.")
				break
			}
			if yes("Turn SSL off and serve plain HTTP on port " + strconv.Itoa(st.Cfg.Port) + "?") {
				st.Cfg.CertFile, st.Cfg.KeyFile = "", ""
				saveAndRestart(st)
				fmt.Println("  Now: " + bold(siteURL(st)))
			}
		case "5":
			if p := readPlainPass(dir); p != "" {
				fmt.Println("  Username: " + st.Cfg.User)
				fmt.Println("  Password: " + bold(p))
			} else {
				fmt.Println("  The password was changed by you, so it is not stored in readable form.")
			}
			if yes("Reset it to a new random password?") {
				p := randPassword(14)
				setPassword(st, p)
				writePlainPass(dir, p)
				saveAndRestart(st)
				fmt.Println("  New password: " + bold(p))
			} else if yes("Set a password of your own?") {
				p := readSecret("New password (min 8 chars)")
				if len(p) < 8 || readSecret("Repeat") != p {
					fmt.Println("  Too short or not matching. Nothing changed.")
					break
				}
				setPassword(st, p)
				writePlainPass(dir, "")
				saveAndRestart(st)
			}
		case "6":
			np := prompt("New admin path", "/"+randHex(6)+"/")
			np = "/" + strings.Trim(strings.TrimSpace(np), "/") + "/"
			if np == "//" || strings.ContainsAny(np, " ?#%\"'<>") || reservedPath(np) {
				fmt.Println("  Invalid or reserved path. Nothing changed.")
				break
			}
			st.Cfg.AdminPath = np
			saveAndRestart(st)
			fmt.Println("  Admin: " + bold(siteURL(st)+np))
		case "7":
			restartService()
		case "8":
			if !githubReachable() {
				fmt.Println("  GitHub is not reachable from this server.")
				break
			}
			if err := selfUpdate(dir, ""); err != nil {
				fmt.Println("  Update failed:", err)
				break
			}
			restartService()
			fmt.Println("  Done. Run `sf` again to open the new menu.")
			return
		case "9":
			if uninstall(dir, true) {
				return
			}
		case "0", "q", "exit":
			clearScreen()
			return
		default:
			continue
		}
		fmt.Print("\n  " + dim("Press Enter to go back…"))
		stdin.ReadString('\n')
	}
}

func reservedPath(p string) bool {
	for _, r := range []string{"/static/", "/api/"} {
		if strings.HasPrefix(p, r) {
			return true
		}
	}
	return false
}

func saveAndRestart(st *Store) {
	if err := st.SaveCfg(); err != nil {
		fmt.Println("  Save failed:", err)
		return
	}
	fmt.Println("  Saved.")
	restartService()
}

func showStatus(st *Store) {
	fmt.Println("  " + bold("Service"))
	if hasSystemd() {
		fmt.Println("    " + svc("is-active", svcName) + dim("  (enabled: "+svc("is-enabled", svcName)+")"))
	}
	fmt.Println("  " + bold("Listening"))
	fmt.Printf("    port %d  %s\n", st.Cfg.Port, map[bool]string{true: "https", false: "http"}[st.Cfg.TLS()])
	fmt.Println("  " + bold("Servers"))
	if list, err := st.Servers(); err == nil {
		ok := 0
		for _, s := range list {
			if s.Status == "installed" {
				ok++
			}
		}
		fmt.Printf("    %d servers, %d with Firefox installed\n", len(list), ok)
	}
	fmt.Println("  " + bold("Data") + "\n    " + st.Dir)
	if hasSystemd() {
		out, _ := exec.Command("journalctl", "-u", svcName, "-n", "5", "--no-pager", "-o", "cat").CombinedOutput()
		if s := strings.TrimSpace(string(out)); s != "" {
			fmt.Println("  " + bold("Last log lines"))
			for _, l := range strings.Split(s, "\n") {
				fmt.Println("    " + dim(l))
			}
		}
	}
}

func sslSetup(st *Store) {
	fmt.Println("  A domain (A record) must already point to this server, and port 80")
	fmt.Println("  must be reachable from the internet for about a minute.")
	fmt.Println()
	dom := strings.ToLower(strings.TrimSpace(prompt("Domain", st.Cfg.Domain)))
	if dom == "" || !hostRe.MatchString(dom) || strings.Contains(dom, ":") {
		fmt.Println("  Invalid domain.")
		return
	}
	email := prompt("Email for expiry notices (optional)", st.Cfg.AcmeEmail)
	stopped := false
	if st.Cfg.Port == 80 && hasSystemd() {
		svc("stop", svcName) // free port 80 for the challenge
		stopped = true
	}
	cf, kf, err := ObtainCert(st.Dir, dom, email, func(f string, a ...any) { fmt.Printf("  "+f+"\n", a...) })
	if err != nil {
		fmt.Println("  " + cRed + "Could not get a certificate:" + cReset)
		fmt.Println("  " + err.Error())
		fmt.Println(dim("  Nothing changed. The site keeps running as before."))
		if stopped {
			svc("start", svcName)
		}
		return
	}
	st.Cfg.Domain, st.Cfg.AcmeEmail, st.Cfg.CertFile, st.Cfg.KeyFile = dom, email, cf, kf
	saveAndRestart(st)
	fmt.Println("  SSL is on: " + bold(siteURL(st)))
	fmt.Println(dim("  Renewed automatically before it expires. Plain http:// on this port redirects to https://."))
}

// ---- update ----

func githubReachable() bool {
	c := &http.Client{Timeout: 6 * time.Second}
	resp, err := c.Head("https://github.com/" + repo + "/releases/latest")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

func httpGet(u string, max int64) ([]byte, error) {
	c := &http.Client{Timeout: 10 * time.Minute}
	resp, err := c.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s: %s", u, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

func sumFor(sums []byte, name string) string {
	for _, l := range strings.Split(string(sums), "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			return strings.ToLower(f[0])
		}
	}
	return ""
}

// selfUpdate replaces the binary from GitHub (src == "") or from another
// Server Firefox server, checking SHA-256. Copies for both arches in
// <dir>/bin are refreshed too, so this server can keep serving clones.
func selfUpdate(dir, src string) error {
	base := "https://github.com/" + repo + "/releases/latest/download"
	if src != "" {
		base = strings.TrimRight(src, "/") + "/dl"
		if !strings.Contains(src, "://") {
			base = "http://" + base
		}
	}
	sums, err := httpGet(base+"/SHA256SUMS", 1<<16)
	if err != nil {
		return err
	}
	target := binPath
	if exe, err := os.Executable(); err == nil {
		if r, err := filepath.EvalSymlinks(exe); err == nil && filepath.Base(r) == binName {
			target = r
		}
	}
	got := 0
	for _, arch := range []string{runtime.GOARCH, otherArch()} {
		name := "server-firefox-linux-" + arch
		want := sumFor(sums, name)
		if want == "" {
			if arch == runtime.GOARCH {
				return errors.New(name + " is not offered there")
			}
			continue
		}
		fmt.Println("  Downloading " + name + " …")
		bin, err := httpGet(base+"/"+name, 200<<20)
		if err != nil {
			if arch == runtime.GOARCH {
				return err
			}
			continue
		}
		h := sha256.Sum256(bin)
		if hex.EncodeToString(h[:]) != want {
			return errors.New("checksum mismatch for " + name)
		}
		_ = os.MkdirAll(filepath.Join(dir, "bin"), 0o755)
		_ = writeAtomic(filepath.Join(dir, "bin", name), bin, 0o755)
		if arch == runtime.GOARCH {
			if err := writeAtomic(target, bin, 0o755); err != nil {
				return err
			}
			got++
		}
	}
	if got == 0 {
		return errors.New("nothing downloaded")
	}
	out, _ := exec.Command(target, "version").Output()
	fmt.Println("  Installed: " + strings.TrimSpace(string(out)))
	return nil
}

func otherArch() string {
	if runtime.GOARCH == "amd64" {
		return "arm64"
	}
	return "amd64"
}

// ---- uninstall ----

func uninstall(dir string, ask bool) bool {
	if ask && !yes("Remove Server Firefox, its service and ALL its data in "+dir+"?") {
		fmt.Println("  Nothing removed.")
		return false
	}
	if hasSystemd() {
		svc("disable", "--now", svcName)
		os.Remove("/etc/systemd/system/" + svcName + ".service")
		svc("daemon-reload")
	}
	os.RemoveAll(dir)
	if l, err := os.Readlink("/usr/local/bin/sf"); err == nil && strings.HasSuffix(l, binName) {
		os.Remove("/usr/local/bin/sf")
	}
	os.Remove(binPath)
	fmt.Println("  Server Firefox removed.")
	return true
}
