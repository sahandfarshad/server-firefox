package main

import (
	crand "crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

var version = "0.2.1"

const (
	repo    = "sahandfarshad/server-firefox"
	binName = "server-firefox"
	binPath = "/usr/local/bin/server-firefox"
	svcName = "server-firefox"
)

func dataDir() string {
	if d := os.Getenv("SF_DIR"); d != "" {
		return d
	}
	return "/var/lib/server-firefox"
}

func main() {
	dir := dataDir()
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	args := os.Args[min(2, len(os.Args)):]
	switch cmd {
	case "serve":
		serve(dir)
	case "init":
		port := 0
		for i := 0; i < len(args); i++ {
			if args[i] == "--port" && i+1 < len(args) {
				port, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
		initDefaults(dir, port)
	case "", "menu":
		if cmd == "" && !term.IsTerminal(int(os.Stdin.Fd())) {
			serve(dir)
			return
		}
		runMenu(dir)
	case "freeport":
		fmt.Println(suggestPort())
	case "portfree":
		p, err := strconv.Atoi(strings.TrimSpace(strings.Join(args, "")))
		if err != nil || p < 1 || p > 65535 || !portFree(p) {
			os.Exit(1)
		}
	case "info":
		st := mustStore(dir)
		printInfo(st)
	case "update":
		if err := selfUpdate(dir, ""); err != nil {
			fatal("update failed: " + err.Error())
		}
	case "uninstall":
		uninstall(dir, true)
	case "version", "-v", "--version":
		fmt.Println("Server Firefox", version)
	default:
		fmt.Println("usage: server-firefox [menu|serve|init|info|update|uninstall|version]")
		os.Exit(2)
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func mustStore(dir string) *Store {
	st, err := OpenStore(dir)
	if err != nil {
		fatal(err.Error())
	}
	return st
}

func serve(dir string) {
	st := mustStore(dir)
	if st.Cfg.Port == 0 || st.Cfg.PassHash == "" {
		log.Fatal("not initialised yet: run `server-firefox init`")
	}
	srv, err := NewPanel(st)
	if err != nil {
		log.Fatal(err)
	}
	h := srv.Routes()
	addr := fmt.Sprintf(":%d", st.Cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	mk := func(h http.Handler) *http.Server {
		// no Read/WriteTimeout: large uploads and slow downloads must finish
		return &http.Server{Handler: h, ReadHeaderTimeout: 15 * time.Second, IdleTimeout: 2 * time.Minute}
	}
	if st.Cfg.TLS() {
		if _, err := os.Stat(st.Cfg.CertFile); err == nil {
			r := &certReloader{certFile: st.Cfg.CertFile, keyFile: st.Cfg.KeyFile}
			tl, pl := splitPort(ln, &tls.Config{GetCertificate: r.Get, MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}})
			go mk(redirectHandler(st.Cfg.Domain)).Serve(pl)
			go renewLoop(st)
			log.Printf("Server Firefox %s on %s", version, siteURL(st))
			log.Fatal(mk(h).Serve(tl))
		}
		log.Printf("certificate %s not found, serving plain HTTP", st.Cfg.CertFile)
	}
	log.Printf("Server Firefox %s on %s", version, siteURL(st))
	log.Fatal(mk(h).Serve(ln))
}

// renewLoop renews our own Let's Encrypt certificate 30 days before expiry.
func renewLoop(st *Store) {
	for {
		time.Sleep(time.Minute)
		if exp, err := certExpiry(st.Cfg.CertFile); err == nil && time.Until(exp) < 30*24*time.Hour && st.Cfg.Domain != "" {
			if _, _, err := ObtainCert(st.Dir, st.Cfg.Domain, st.Cfg.AcmeEmail, log.Printf); err != nil {
				log.Printf("certificate renewal failed: %v", err)
			}
		}
		time.Sleep(12 * time.Hour)
	}
}

func siteURL(st *Store) string {
	c := st.Cfg
	if c.TLS() && c.Domain != "" {
		return fmt.Sprintf("https://%s%s", c.Domain, portSuffix("https", c.Port))
	}
	host := c.PublicHost
	if host == "" {
		host = "SERVER_IP"
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("http://%s%s", host, portSuffix("http", c.Port))
}

func portSuffix(scheme string, p int) string {
	if (scheme == "http" && p == 80) || (scheme == "https" && p == 443) {
		return ""
	}
	return ":" + strconv.Itoa(p)
}

func suggestPort() int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 300; i++ {
		p := 20000 + r.Intn(40000)
		if portFree(p) {
			return p
		}
	}
	return 0
}

func randPassword(n int) string {
	const chars = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	raw := make([]byte, n)
	_, _ = crand.Read(raw)
	for i := range raw {
		raw[i] = chars[int(raw[i])%len(chars)]
	}
	return string(raw)
}

func setPassword(st *Store, pass string) {
	h, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	st.Cfg.PassHash = string(h)
}

func publicIP() string {
	c := &http.Client{Timeout: 4 * time.Second}
	for _, u := range []string{"https://api.ipify.org", "https://ifconfig.me/ip", "https://icanhazip.com"} {
		resp, err := c.Get(u)
		if err != nil {
			continue
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		if ip := strings.TrimSpace(string(b)); resp.StatusCode == 200 && net.ParseIP(ip) != nil {
			return ip
		}
	}
	// offline (e.g. during a shutdown): best local guess
	var private string
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if !ok || n.IP.IsLoopback() || n.IP.IsLinkLocalUnicast() || n.IP.To4() == nil {
			continue
		}
		if !n.IP.IsPrivate() {
			return n.IP.String()
		}
		if private == "" {
			private = n.IP.String()
		}
	}
	return private
}

// initDefaults creates working settings once: port, secret admin path,
// random password. An existing install keeps its settings (port may change).
func initDefaults(dir string, port int) {
	st := mustStore(dir)
	fresh := st.Cfg.PassHash == ""
	if fresh {
		pass := randPassword(14)
		st.Cfg.User = "admin"
		setPassword(st, pass)
		st.Cfg.Secret = randHex(32)
		st.Cfg.AdminPath = "/" + randHex(6) + "/"
		writePlainPass(dir, pass)
	}
	if port > 0 {
		st.Cfg.Port = port
	}
	if st.Cfg.Port == 0 {
		st.Cfg.Port = suggestPort()
	}
	if st.Cfg.PublicHost == "" {
		st.Cfg.PublicHost = publicIP()
	}
	if err := st.SaveCfg(); err != nil {
		fatal(err.Error())
	}
	if !fresh {
		fmt.Println("Existing settings kept.")
	}
	printInfo(st)
}

func printInfo(st *Store) {
	fmt.Println()
	fmt.Println("  " + cGold + "✦ Server Firefox" + cReset + dim("  v"+version))
	fmt.Println("  " + dim(strings.Repeat("─", 49)))
	fmt.Println("  Panel:     " + siteURL(st) + st.Cfg.AdminPath)
	fmt.Println("  Username:  " + st.Cfg.User)
	if p := readPlainPass(st.Dir); p != "" {
		fmt.Println("  Password:  " + p)
	} else {
		fmt.Println("  Password:  " + dim("(changed by you; reset it from the menu if lost)"))
	}
	fmt.Println("  " + dim(strings.Repeat("─", 49)))
	fmt.Println("  Menu any time: " + bold("sf") + dim("  (or server-firefox)"))
	fmt.Println()
}
