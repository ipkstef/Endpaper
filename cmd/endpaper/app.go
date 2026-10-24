package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ipkstef/endpaper/internal/appreg"
	"github.com/ipkstef/endpaper/internal/atomicfile"
)

const (
	// appID is our app's id in appreg.db and in ui/config.xml.
	appID = "io.github.ipkstef.endpaper"
	// uiDir holds the page. Not in the state folder: Mesquite may run as
	// another user (UNVERIFIED), so the page is world-readable; the state
	// folder (token) is not.
	uiDir    = "/var/local/mesquite/endpaper"
	mesquite = "/usr/bin/mesquite"
)

//go:embed ui
var uiFiles embed.FS

// installUI writes the page and its config.js (port, key, pid).
func installUI(port int, key string) error {
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		return err
	}
	_ = os.Chmod(uiDir, 0o755)
	err := fs.WalkDir(uiFiles, "ui", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := uiFiles.ReadFile(p)
		if err != nil {
			return err
		}
		return atomicfile.Write(filepath.Join(uiDir, filepath.Base(p)), b, 0o644)
	})
	if err != nil {
		return err
	}
	cfg := fmt.Sprintf("var HC = {port: %d, key: %q, pid: %d, version: %q};\n", port, key, os.Getpid(), version)
	return atomicfile.Write(filepath.Join(uiDir, "config.js"), []byte(cfg), 0o644)
}

var cfgRe = regexp.MustCompile(`pid: (\d+), version: "([^"]*)"`)

// uiDaemon reads the pid and version the running daemon wrote.
func uiDaemon() (pid int, ver string) {
	b, err := os.ReadFile(filepath.Join(uiDir, "config.js"))
	if err != nil {
		return 0, ""
	}
	m := cfgRe.FindSubmatch(b)
	if m == nil {
		return 0, ""
	}
	pid, _ = strconv.Atoi(string(m[1]))
	return pid, string(m[2])
}

// launch is the library tap ("Endpaper.sh"): make sure this version of
// the daemon runs, then open our page. If the page cannot open (no
// Mesquite, appreg or LIPC on this Kindle), fall back to text on the
// screen: sign in, or the status.
func (a *app) launch(ctx context.Context) error {
	if err := a.ensureDaemon(); err != nil {
		log.Printf("launch: daemon: %v", err)
	}
	err := a.openPage(ctx)
	if err == nil {
		return nil
	}
	log.Printf("launch: page: %v (text screen instead)", err)
	if _, terr := a.store.Load(); terr != nil {
		return a.login(ctx)
	}
	return a.status()
}

// ensureDaemon starts the daemon when it is not running or runs another
// version (an update was copied over), and waits until its page config is
// written.
func (a *app) ensureDaemon() error {
	pid := a.running()
	if p, v := uiDaemon(); pid != 0 && p == pid && v == version {
		return nil
	}
	if err := a.startDaemon(); err != nil {
		return err
	}
	// The new daemon may wait for an old one to stop (up to ~25 s).
	for i := 0; i < 120; i++ {
		time.Sleep(250 * time.Millisecond)
		if p, v := uiDaemon(); p != 0 && p == a.running() && v == version {
			return nil
		}
	}
	return errors.New("daemon did not start in 30 s")
}

func (a *app) openPage(ctx context.Context) error {
	if _, err := os.Stat(mesquite); err != nil {
		return err
	}
	if _, err := exec.LookPath("lipc-set-prop"); err != nil {
		return err
	}
	if p, _ := uiDaemon(); p == 0 || p != a.running() {
		return errors.New("background sync is not running")
	}
	changed, err := appreg.Register(appreg.DefaultPath, appID, map[string]string{
		"command":              mesquite + " -l " + appID + " -c file://" + uiDir + "/",
		"lipcId":               appID,
		"supportedOrientation": "U",
	}, filepath.Join(a.stateDir, "appreg.db.bak"))
	if err != nil {
		return err
	}
	if changed {
		log.Printf("launch: registered %s in appreg.db", appID)
	}
	// Close a page that is still open (also an older version's): "start"
	// would only bring it back with its old code and connection. A fresh
	// page reads the current config.js.
	if n := closePage("/proc"); n > 0 {
		log.Printf("launch: closed %d open page process(es)", n)
	}
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(c, "lipc-set-prop", "com.lab126.appmgrd", "start", "app://"+appID).CombinedOutput(); err != nil {
		return fmt.Errorf("appmgrd start: %v %s", err, out)
	}
	// For the test log: which window is in front a moment later.
	time.Sleep(3 * time.Second)
	log.Printf("launch: page started, active window %q", lipcGet(ctx, "com.lab126.winmgr", "getActiveAppTitle"))
	return nil
}

// pagePids returns the processes that run our page: Mesquite started with
// "-l <appID>" (the command we register in appreg.db).
func pagePids(proc string) []int {
	dirs, _ := os.ReadDir(proc)
	var pids []int
	for _, d := range dirs {
		pid, err := strconv.Atoi(d.Name())
		if err != nil || pid == os.Getpid() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(proc, d.Name(), "cmdline"))
		if err != nil {
			continue
		}
		args := strings.Split(strings.TrimRight(string(b), "\x00"), "\x00")
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "-l" && args[i+1] == appID && strings.HasSuffix(args[0], "mesquite") {
				pids = append(pids, pid)
				break
			}
		}
	}
	return pids
}

// closePage stops open page processes (TERM, then KILL after 3 s) and
// returns how many it found.
func closePage(proc string) int {
	pids := pagePids(proc)
	n := len(pids)
	for _, p := range pids {
		_ = syscall.Kill(p, syscall.SIGTERM)
	}
	for i := 0; i < 15 && len(pids) > 0; i++ {
		time.Sleep(200 * time.Millisecond)
		alive := pids[:0]
		for _, p := range pids {
			if syscall.Kill(p, 0) == nil {
				alive = append(alive, p)
			}
		}
		pids = alive
	}
	for _, p := range pids {
		_ = syscall.Kill(p, syscall.SIGKILL)
	}
	return n
}
