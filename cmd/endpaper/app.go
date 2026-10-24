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
	"time"

	"github.com/ipkstef/kindle-hardcover-bridge/internal/appreg"
	"github.com/ipkstef/kindle-hardcover-bridge/internal/atomicfile"
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
	for i := 0; i < 40; i++ {
		time.Sleep(250 * time.Millisecond)
		if p, v := uiDaemon(); p != 0 && p == a.running() && v == version {
			return nil
		}
	}
	return errors.New("daemon did not start in 10 s")
}

func (a *app) openPage(ctx context.Context) error {
	if _, err := os.Stat(mesquite); err != nil {
		return err
	}
	if _, err := exec.LookPath("lipc-set-prop"); err != nil {
		return err
	}
	if p, _ := uiDaemon(); p == 0 {
		return errors.New("page not installed")
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
