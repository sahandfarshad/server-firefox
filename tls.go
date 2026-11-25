package main

// SSL: a Let's Encrypt certificate via the HTTP-01 challenge (port 80 is
// borrowed for a minute), then HTTPS on the same port the user chose. Plain
// HTTP requests that hit the HTTPS port are redirected instead of failing.

import (
	"bufio"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme"
)

func certExpiry(certFile string) (time.Time, error) {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return time.Time{}, err
	}
	b, _ := pem.Decode(data)
	if b == nil {
		return time.Time{}, errors.New("not a PEM certificate")
	}
	c, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return time.Time{}, err
	}
	return c.NotAfter, nil
}

type certReloader struct {
	mu                sync.Mutex
	certFile, keyFile string
	mtime             time.Time
	cert              *tls.Certificate
}

func (r *certReloader) Get(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := os.Stat(r.certFile)
	if err == nil && (r.cert == nil || st.ModTime().After(r.mtime)) {
		c, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
		if err == nil {
			r.cert, r.mtime = &c, st.ModTime()
		} else if r.cert == nil {
			return nil, err
		}
	}
	if r.cert == nil {
		return nil, errors.New("no certificate")
	}
	return r.cert, nil
}

func portFree(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	l.Close()
	return true
}

// ObtainCert gets a certificate for domain and stores it in <dir>/certs/<domain>/.
func ObtainCert(dataDir, domain, email string, logf func(string, ...any)) (string, string, error) {
	if l, err := net.Listen("tcp", ":80"); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			return "", "", errors.New("binding port 80 needs root; run the menu as root")
		}
		return "", "", errors.New("port 80 is busy (nginx, apache or another program). Let's Encrypt needs it free for about a minute; stop that program and try again")
	} else {
		l.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	accKey, err := loadOrCreateKey(filepath.Join(dataDir, "acme-account.key"))
	if err != nil {
		return "", "", err
	}
	dirURL := acme.LetsEncryptURL
	if u := os.Getenv("SP_ACME_DIRECTORY"); u != "" { // e.g. the staging endpoint for tests
		dirURL = u
	}
	client := &acme.Client{Key: accKey, DirectoryURL: dirURL}
	acct := &acme.Account{}
	if email != "" {
		acct.Contact = []string{"mailto:" + email}
	}
	if _, err := client.Register(ctx, acct, acme.AcceptTOS); err != nil && !errors.Is(err, acme.ErrAccountAlreadyExists) {
		return "", "", fmt.Errorf("cannot reach Let's Encrypt (needs internet): %w", err)
	}
	tokens := map[string]string{}
	var tmu sync.Mutex
	srv := &http.Server{Addr: ":80", ReadHeaderTimeout: 10 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tmu.Lock()
		body, ok := tokens[r.URL.Path]
		tmu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	})}
	go srv.ListenAndServe()
	defer srv.Close()

	logf("Requesting a certificate for %s …", domain)
	order, err := client.AuthorizeOrder(ctx, acme.DomainIDs(domain))
	if err != nil {
		return "", "", fmt.Errorf("order: %w", err)
	}
	for _, zurl := range order.AuthzURLs {
		z, err := client.GetAuthorization(ctx, zurl)
		if err != nil {
			return "", "", err
		}
		if z.Status == acme.StatusValid {
			continue
		}
		var chal *acme.Challenge
		for _, c := range z.Challenges {
			if c.Type == "http-01" {
				chal = c
			}
		}
		if chal == nil {
			return "", "", errors.New("no http-01 challenge offered")
		}
		resp, err := client.HTTP01ChallengeResponse(chal.Token)
		if err != nil {
			return "", "", err
		}
		tmu.Lock()
		tokens[client.HTTP01ChallengePath(chal.Token)] = resp
		tmu.Unlock()
		if _, err := client.Accept(ctx, chal); err != nil {
			return "", "", err
		}
		logf("Waiting for Let's Encrypt to check http://%s/ …", domain)
		if _, err := client.WaitAuthorization(ctx, z.URI); err != nil {
			return "", "", fmt.Errorf("validation failed. Does %s point to this server, and is port 80 open in the firewall? (%v)", domain, err)
		}
	}
	if order, err = client.WaitOrder(ctx, order.URI); err != nil {
		return "", "", err
	}
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{DNSNames: []string{domain}}, key)
	if err != nil {
		return "", "", err
	}
	ders, _, err := client.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(dataDir, "certs", domain)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", err
	}
	var chain []byte
	for _, d := range ders {
		chain = append(chain, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: d})...)
	}
	kb, _ := x509.MarshalECPrivateKey(key)
	cf, kf := filepath.Join(dir, "fullchain.pem"), filepath.Join(dir, "privkey.pem")
	if err := os.WriteFile(kf, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}), 0o600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(cf, chain, 0o644); err != nil {
		return "", "", err
	}
	logf("Certificate saved: %s", cf)
	return cf, kf, nil
}

func loadOrCreateKey(p string) (crypto.Signer, error) {
	if data, err := os.ReadFile(p); err == nil {
		if b, _ := pem.Decode(data); b != nil {
			return x509.ParseECPrivateKey(b.Bytes)
		}
	}
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	b, _ := x509.MarshalECPrivateKey(k)
	if err := os.WriteFile(p, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b}), 0o600); err != nil {
		return nil, err
	}
	return k, nil
}

// ---- one port, TLS or a redirect ----

type peekConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *peekConn) Read(b []byte) (int, error) { return c.r.Read(b) }

// splitPort accepts on one listener and sorts each connection by its first
// byte: TLS handshakes go to the returned TLS listener, anything else to the
// plain listener (which only redirects to https). Sorting happens per
// connection, so a slow client never holds up the others.
func splitPort(ln net.Listener, tlsCfg *tls.Config) (net.Listener, net.Listener) {
	tl := &chanListener{ch: make(chan net.Conn, 64), addr: ln.Addr()}
	pl := &chanListener{ch: make(chan net.Conn, 64), addr: ln.Addr()}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				close(tl.ch)
				close(pl.ch)
				return
			}
			go func(c net.Conn) {
				_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
				br := bufio.NewReader(c)
				b, err := br.Peek(1)
				_ = c.SetReadDeadline(time.Time{})
				if err != nil {
					c.Close()
					return
				}
				pc := &peekConn{c, br}
				if b[0] == 0x16 {
					tl.ch <- tls.Server(pc, tlsCfg)
				} else {
					pl.ch <- pc
				}
			}(c)
		}
	}()
	return tl, pl
}

type chanListener struct {
	ch   chan net.Conn
	addr net.Addr
}

func (l *chanListener) Accept() (net.Conn, error) {
	c, ok := <-l.ch
	if !ok {
		return nil, errors.New("closed")
	}
	return c, nil
}
func (l *chanListener) Close() error   { return nil }
func (l *chanListener) Addr() net.Addr { return l.addr }

func redirectHandler(domain string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if domain != "" {
			if _, p, err := net.SplitHostPort(r.Host); err == nil {
				host = domain + ":" + p
			} else {
				host = domain
			}
		}
		target := "https://" + strings.TrimSuffix(host, ":443") + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}
