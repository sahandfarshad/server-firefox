package main

// Data layout (default /var/lib/server-firefox, override with SF_DIR):
//
//	config.json   panel settings: port, admin path, login hash, SSL
//	servers.json  managed servers; SSH and Firefox passwords AES-GCM encrypted
//	secret.key    32-byte key for that encryption (root only)
//	certs/        Let's Encrypt certificate of the panel itself

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	Port       int    `json:"port"`
	User       string `json:"user"`
	PassHash   string `json:"passHash"`
	Secret     string `json:"secret"`
	AdminPath  string `json:"adminPath"`
	PublicHost string `json:"publicHost"`
	Domain     string `json:"domain,omitempty"`
	AcmeEmail  string `json:"acmeEmail,omitempty"`
	CertFile   string `json:"certFile,omitempty"`
	KeyFile    string `json:"keyFile,omitempty"`
}

func (c *Config) TLS() bool { return c.CertFile != "" && c.KeyFile != "" }

type Server struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	SSHPort    int    `json:"sshPort"`
	SSHUser    string `json:"sshUser"`
	SSHPass    string `json:"sshPass"` // encrypted
	Domain     string `json:"domain"`
	WebUser    string `json:"webUser"`
	WebPass    string `json:"webPass"` // encrypted
	Resolution string `json:"resolution"`
	HostKey    string `json:"hostKey,omitempty"` // SSH fingerprint pinned on first connect
	Status     string `json:"status"`            // new | installing | installed | failed | removed
	LastError  string `json:"lastError,omitempty"`
	Updated    string `json:"updated,omitempty"`
}

type Store struct {
	Dir string
	Cfg Config
	mu  sync.Mutex
	key []byte
}

func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &Store{Dir: dir}
	if b, err := os.ReadFile(filepath.Join(dir, "config.json")); err == nil {
		if err := json.Unmarshal(b, &s.Cfg); err != nil {
			return nil, errors.New("config.json is damaged: " + err.Error())
		}
	}
	kp := filepath.Join(dir, "secret.key")
	k, err := os.ReadFile(kp)
	if err != nil || len(k) != 32 {
		k = make([]byte, 32)
		if _, err := rand.Read(k); err != nil {
			return nil, err
		}
		if err := writeAtomic(kp, k, 0o600); err != nil {
			return nil, err
		}
	}
	s.key = k
	return s, nil
}

func writeAtomic(p string, data []byte, perm os.FileMode) error {
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (s *Store) SaveCfg() error {
	b, _ := json.MarshalIndent(s.Cfg, "", "  ")
	return writeAtomic(filepath.Join(s.Dir, "config.json"), b, 0o600)
}

func (s *Store) Servers() ([]*Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadServers()
}

func (s *Store) loadServers() ([]*Server, error) {
	var list []*Server
	b, err := os.ReadFile(filepath.Join(s.Dir, "servers.json"))
	if errors.Is(err, os.ErrNotExist) {
		return list, nil
	}
	if err != nil {
		return nil, err
	}
	return list, json.Unmarshal(b, &list)
}

// UpdateServers loads, lets fn change the list, and saves it, under one lock.
func (s *Store) UpdateServers(fn func([]*Server) []*Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.loadServers()
	if err != nil {
		return err
	}
	list = fn(list)
	b, _ := json.MarshalIndent(list, "", "  ")
	return writeAtomic(filepath.Join(s.Dir, "servers.json"), b, 0o600)
}

func (s *Store) Server(id string) *Server {
	list, _ := s.Servers()
	for _, x := range list {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (s *Store) SaveServer(sv *Server) error {
	return s.UpdateServers(func(l []*Server) []*Server {
		for i, x := range l {
			if x.ID == sv.ID {
				l[i] = sv
				return l
			}
		}
		return append(l, sv)
	})
}

func (s *Store) Encrypt(plain string) string {
	if plain == "" {
		return ""
	}
	blk, _ := aes.NewCipher(s.key)
	g, _ := cipher.NewGCM(blk)
	nonce := make([]byte, g.NonceSize())
	_, _ = rand.Read(nonce)
	return base64.StdEncoding.EncodeToString(g.Seal(nonce, nonce, []byte(plain), nil))
}

func (s *Store) Decrypt(enc string) string {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil || enc == "" {
		return ""
	}
	blk, _ := aes.NewCipher(s.key)
	g, _ := cipher.NewGCM(blk)
	if len(raw) < g.NonceSize() {
		return ""
	}
	out, err := g.Open(nil, raw[:g.NonceSize()], raw[g.NonceSize():], nil)
	if err != nil {
		return ""
	}
	return string(out)
}

func randHex(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}
