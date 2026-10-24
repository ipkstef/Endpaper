package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"rsc.io/qr"

	"github.com/ipkstef/endpaper/internal/daemon"
)

// api is the daemon's local HTTP interface for our Mesquite page. It
// listens on 127.0.0.1 only, on a free port; the port and a random key are
// written to the page's config.js at each daemon start. Each request must
// carry the key (?k=), so a web page in the Kindle browser cannot use it.
type api struct {
	a         *app
	key       string
	fire      func(string)
	stop      func()
	uninstall func() // asks the main loop to uninstall
	importAll *atomic.Bool
	upd       *updates // nil: no update check

	seen atomic.Bool // first request from the page logged

	mu       sync.Mutex
	login    *loginState
	user     string
	userBusy bool
	userTry  time.Time // last user name fetch (wall clock)
}

type loginState struct {
	Active  bool      `json:"active"`
	Code    string    `json:"code,omitempty"`
	URL     string    `json:"url,omitempty"`
	Expires time.Time `json:"expires,omitempty"`
	Err     string    `json:"error,omitempty"`
	qrText  string
	cancel  context.CancelFunc
}

// userRetry: wait this long before fetching the user name again.
const userRetry = 5 * time.Minute

func newKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// serve starts the server and returns its port.
func (s *api) serve(ctx context.Context) (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handle(s.status))
	mux.HandleFunc("/api/qr.png", s.qr)
	for name, fn := range map[string]func(context.Context) (string, error){
		"login":        s.startLogin,
		"login-cancel": s.cancelLogin,
		"sync":         s.syncNow,
		"import":       s.importOld,
		"check":        s.check,
		"log":          s.saveLog,
		"logout":       s.logout,
		"stop":         s.stopSync,
		"uninstall":    s.uninstallNow,
	} {
		fn := fn
		mux.HandleFunc("/api/"+name, s.handle(func(ctx context.Context) (any, error) {
			msg, err := fn(ctx)
			return map[string]string{"message": msg}, err
		}))
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	go func() { _ = srv.Serve(ln) }()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func (s *api) authorized(r *http.Request) bool {
	return subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("k")), []byte(s.key)) == 1
}

// handle wraps a handler: key check, JSON out. The page is a file:// page,
// so the answer needs an open CORS header (the key protects it).
func (s *api) handle(fn func(context.Context) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if !s.authorized(r) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"bad key"}`))
			return
		}
		if !s.seen.Swap(true) {
			log.Printf("page api: first request from the page (%s)", r.URL.Path)
		}
		ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancel()
		out, err := fn(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			out = map[string]string{"error": err.Error()}
		}
		_ = json.NewEncoder(w).Encode(out)
	}
}

type statusOut struct {
	Version    string      `json:"version"`
	SignedIn   bool        `json:"signedIn"`
	User       string      `json:"user,omitempty"`
	LastCheck  string      `json:"lastCheck,omitempty"`
	LastAt     int64       `json:"lastAt,omitempty"` // Unix seconds, for "sync now" done
	LastResult string      `json:"lastResult,omitempty"`
	Pending    int         `json:"pending"`
	Login      *loginState `json:"login,omitempty"`
	Update     string      `json:"update,omitempty"` // newer release tag
	UpdateURL  string      `json:"updateURL,omitempty"`
}

func (s *api) status(context.Context) (any, error) {
	out := statusOut{Version: version}
	if s.upd != nil {
		if out.Update = s.upd.available(version); out.Update != "" {
			out.UpdateURL = releasesURL
		}
	}
	_, err := s.a.store.Load()
	out.SignedIn = err == nil
	if db := s.a.stateDB(); db != nil {
		last, res, pending := daemon.ReadStatus(db)
		if !last.IsZero() {
			out.LastCheck = last.Local().Format("Jan 2, 15:04")
			out.LastAt = last.Unix()
		}
		out.LastResult, out.Pending = res, pending
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.login != nil && (s.login.Active || s.login.Err != "") {
		l := *s.login
		out.Login = &l
	}
	out.User = s.user
	// The name needs the network: at most one try per userRetry (else an
	// offline Kindle tries every few seconds while the page is open).
	if out.SignedIn && s.user == "" && !s.userBusy && time.Since(s.userTry) >= userRetry {
		s.userBusy, s.userTry = true, time.Now().Round(0)
		go s.fetchUser()
	}
	return out, nil
}

// fetchUser reads the user name once, in the background (status must not
// wait for the network).
func (s *api) fetchUser() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	me, err := s.a.client().Me(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userBusy = false
	if err == nil {
		s.user = "@" + me.Username
	}
}

func (s *api) startLogin(context.Context) (string, error) {
	if s.a.oauth.ClientID == "" {
		return "", errors.New("no client ID in this build (client_id.txt missing)")
	}
	s.mu.Lock()
	if s.login != nil && s.login.Active {
		s.mu.Unlock()
		return "Sign-in already started.", nil
	}
	lctx, cancel := context.WithCancel(context.Background())
	l := &loginState{Active: true, cancel: cancel}
	s.login = l
	s.mu.Unlock()

	// Each sign-in only changes its own state (l): after Cancel and a new
	// Sign in, the old one must not touch the new one.
	dc, err := s.a.oauth.RequestDeviceCode(lctx, s.a.scope)
	if err != nil {
		s.loginDone(l, fmt.Errorf("could not get a code: %w", err))
		return "", err
	}
	log.Printf("login: device code issued, user code %s, expires in %ds", dc.UserCode, dc.ExpiresIn)
	s.mu.Lock()
	l.Code = dc.UserCode
	l.URL = strings.TrimPrefix(dc.VerificationURI, "https://")
	l.Expires = time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)
	l.qrText = dc.VerificationURIComplete
	if l.qrText == "" {
		l.qrText = dc.VerificationURI
	}
	s.mu.Unlock()
	go func() {
		defer cancel()
		tok, err := s.a.oauth.PollToken(lctx, dc)
		if err == nil {
			err = s.a.store.Save(tok)
		}
		if err == nil {
			s.a.client().ForgetMe()
			if me, merr := s.a.client().Me(lctx); merr == nil {
				s.mu.Lock()
				s.user = "@" + me.Username
				s.mu.Unlock()
				log.Printf("login: OK as %s (id %d)", me.Username, me.ID)
			}
			s.fire("signed in")
		}
		s.loginDone(l, err)
	}()
	return "", nil
}

// loginDone ends sign-in l. Only the current sign-in is shown on the page.
func (s *api) loginDone(l *loginState, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l.Active = false
	l.Code, l.URL, l.qrText = "", "", ""
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("login: %v", err)
		l.Err = err.Error()
	} else {
		l.Err = ""
	}
}

func (s *api) cancelLogin(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.login != nil && s.login.cancel != nil {
		s.login.cancel()
	}
	s.login = nil
	return "", nil
}

func (s *api) qr(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	s.mu.Lock()
	text := ""
	if s.login != nil {
		text = s.login.qrText
	}
	s.mu.Unlock()
	if text == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	c, err := qr.Encode(text, qr.M)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Scale = 6
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(c.PNG())
}

func (s *api) syncNow(context.Context) (string, error) {
	if _, err := s.a.store.Load(); err != nil {
		return "", errors.New("sign in first")
	}
	s.fire("sync now")
	return "Checking your books now.", nil
}

func (s *api) importOld(context.Context) (string, error) {
	if _, err := s.a.store.Load(); err != nil {
		return "", errors.New("sign in first")
	}
	s.importAll.Store(true)
	s.fire("import")
	return "Sending all highlights and notes. This can take a few minutes.", nil
}

func (s *api) check(ctx context.Context) (string, error) {
	lines, _ := s.a.checks(ctx)
	return strings.Join(lines, "\n"), nil
}

func (s *api) saveLog(context.Context) (string, error) {
	p, err := s.a.saveLogFile()
	if err != nil {
		return "", err
	}
	return "Log saved to " + strings.TrimPrefix(p, usbRoot+"/") + ". Connect USB to copy it.", nil
}

func (s *api) logout(ctx context.Context) (string, error) {
	_, _ = s.cancelLogin(ctx)
	// Same locks as a token refresh: a refresh running now must not save a
	// new token after the delete.
	s.a.tokMu.Lock()
	defer s.a.tokMu.Unlock()
	unlock, err := s.a.store.Lock()
	if err != nil {
		return "", err
	}
	defer unlock()
	s.a.unsaved = nil
	if err := s.a.store.Delete(); err != nil {
		return "", err
	}
	s.a.client().ForgetMe()
	s.mu.Lock()
	s.user = ""
	s.mu.Unlock()
	log.Printf("logout: token deleted")
	return "Signed out.", nil
}

func (s *api) stopSync(context.Context) (string, error) {
	go func() {
		time.Sleep(300 * time.Millisecond) // let the answer go out
		s.stop()
	}()
	return "Sync stopped. Tap Endpaper in your library to start it again.", nil
}

func (s *api) uninstallNow(context.Context) (string, error) {
	go func() {
		time.Sleep(300 * time.Millisecond) // let the answer go out
		s.uninstall()
	}()
	return "Endpaper was removed from this Kindle.", nil
}
