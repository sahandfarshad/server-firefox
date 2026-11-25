package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

//go:embed remote.sh
var remoteScript string

// Job is one running or finished action on a server, with its live log.
type Job struct {
	mu     sync.Mutex
	Action string
	Lines  []string
	Done   bool
	OK     bool
}

func (j *Job) add(s string) {
	j.mu.Lock()
	j.Lines = append(j.Lines, s)
	if len(j.Lines) > 5000 {
		j.Lines = j.Lines[len(j.Lines)-5000:]
	}
	j.mu.Unlock()
}

func (j *Job) snapshot(from int) ([]string, int, bool, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if from < 0 || from > len(j.Lines) {
		from = 0
	}
	return append([]string(nil), j.Lines[from:]...), len(j.Lines), j.Done, j.OK
}

type Runner struct {
	st   *Store
	mu   sync.Mutex
	jobs map[string]*Job
}

func NewRunner(st *Store) *Runner { return &Runner{st: st, jobs: map[string]*Job{}} }

func (r *Runner) Job(id string) *Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.jobs[id]
}

// Start runs action (install | status | restart | uninstall) on server id.
func (r *Runner) Start(id, action string) error {
	sv := r.st.Server(id)
	if sv == nil {
		return errors.New("server not found")
	}
	r.mu.Lock()
	if j := r.jobs[id]; j != nil && !j.Done {
		r.mu.Unlock()
		return errors.New("another action is still running on this server")
	}
	j := &Job{Action: action}
	r.jobs[id] = j
	r.mu.Unlock()
	if action == "install" {
		sv.Status, sv.LastError = "installing", ""
		_ = r.st.SaveServer(sv)
	}
	go r.run(sv, j)
	return nil
}

func (r *Runner) run(sv *Server, j *Job) {
	j.add(fmt.Sprintf("▶ %s on %s (%s) — %s", j.Action, sv.Name, sv.Host, time.Now().Format("2006-01-02 15:04:05")))
	err := r.exec(sv, j)
	if j.Action == "install" || j.Action == "uninstall" {
		if s := r.st.Server(sv.ID); s != nil {
			sv = s
		}
		switch {
		case err != nil:
			sv.Status, sv.LastError = "failed", err.Error()
		case j.Action == "install":
			sv.Status, sv.LastError = "installed", ""
		default:
			sv.Status, sv.LastError = "removed", ""
		}
		sv.Updated = time.Now().Format("2006-01-02 15:04")
		_ = r.st.SaveServer(sv)
	}
	j.mu.Lock()
	j.Done, j.OK = true, err == nil
	j.mu.Unlock()
	if err != nil {
		j.add("✖ " + err.Error())
	} else {
		j.add("✔ done")
	}
}

func (r *Runner) exec(sv *Server, j *Job) error {
	pass := r.st.Decrypt(sv.SSHPass)
	auth := []ssh.AuthMethod{
		ssh.Password(pass),
		ssh.KeyboardInteractive(func(_, _ string, qs []string, _ []bool) ([]string, error) {
			a := make([]string, len(qs))
			for i := range a {
				a[i] = pass
			}
			return a, nil
		}),
	}
	var newKey string
	cfg := &ssh.ClientConfig{
		User: sv.SSHUser,
		Auth: auth,
		HostKeyCallback: func(_ string, _ net.Addr, k ssh.PublicKey) error {
			h := sha256.Sum256(k.Marshal())
			fp := "SHA256:" + base64.RawStdEncoding.EncodeToString(h[:])
			if sv.HostKey != "" && sv.HostKey != fp {
				return fmt.Errorf("SSH host key changed (expected %s, got %s); if you reinstalled the server, edit it and clear the host key", sv.HostKey, fp)
			}
			newKey = fp
			return nil
		},
		Timeout: 20 * time.Second,
	}
	addr := net.JoinHostPort(sv.Host, strconv.Itoa(sv.SSHPort))
	j.add("Connecting to " + addr + " …")
	cl, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %v", err)
	}
	defer cl.Close()
	if sv.HostKey == "" && newKey != "" {
		sv.HostKey = newKey
		if s := r.st.Server(sv.ID); s != nil {
			s.HostKey = newKey
			_ = r.st.SaveServer(s)
		}
	}
	j.add("Connected. Host key " + newKey)
	sess, err := cl.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	env := map[string]string{
		"SF_ACTION": j.Action,
		"SF_DOMAIN": sv.Domain,
		"SF_USER":   sv.WebUser,
		"SF_PASS":   r.st.Decrypt(sv.WebPass),
		"SF_RES":    sv.Resolution,
	}
	var head strings.Builder
	for k, v := range env {
		head.WriteString("export " + k + "='" + strings.ReplaceAll(v, "'", `'\''`) + "'\n")
	}
	// Secrets go through stdin, never the command line.
	sess.Stdin = strings.NewReader(head.String() + remoteScript)
	pr := &lineWriter{fn: j.add}
	sess.Stdout, sess.Stderr = pr, pr
	cmd := "bash -s"
	if sv.SSHUser != "root" {
		cmd = "sudo -n bash -s"
	}
	err = sess.Run(cmd)
	pr.flush()
	if err != nil {
		var ee *ssh.ExitError
		if errors.As(err, &ee) {
			return fmt.Errorf("remote step failed (exit %d); see the log above", ee.ExitStatus())
		}
		return err
	}
	return nil
}

type lineWriter struct {
	mu  sync.Mutex
	buf []byte
	fn  func(string)
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	for {
		i := strings.IndexAny(string(w.buf), "\n\r")
		if i < 0 {
			break
		}
		if l := strings.TrimRight(string(w.buf[:i]), " "); l != "" {
			w.fn(l)
		}
		w.buf = w.buf[i+1:]
	}
	return len(p), nil
}

func (w *lineWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) > 0 {
		w.fn(string(w.buf))
		w.buf = nil
	}
}
