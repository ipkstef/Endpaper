// Command homeprobe is a read-only research tool: it collects what is needed
// to understand how the Kindle home screen is built and where its store /
// recommendation content comes from (research for a possible Hardcover home
// screen). It is not part of Endpaper releases.
//
// It reads system files and runs read-only Kindle tools (ps data from /proc,
// lipc-get-prop, lipc-probe, sqlite reads). It writes only to OutDir on the
// USB drive. It changes nothing on the Kindle.
//
// Phases:
//  1. device, mounts, processes, app registry, LIPC services and values;
//  2. file inventory of system and data folders; schema of every SQLite DB;
//     copies of small files whose names look related (home, store, merch,
//     recommendations, widgets, covers) and of the home booklet code;
//  3. a timed watch (watchFor): every file change in the data folders and
//     the window/app in front, while the user goes Home, scrolls, opens the
//     store row, goes to the Library and back.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/ipkstef/endpaper/internal/events"
	"github.com/ipkstef/endpaper/internal/screen"
)

// OutDir and watchFor can be changed with ENDPAPER_PROBE_OUT and
// ENDPAPER_PROBE_WATCH (a Go duration) for a test run off the Kindle.
var (
	OutDir   = envOr("ENDPAPER_PROBE_OUT", "/mnt/us/endpaper-probe")
	watchFor = durOr("ENDPAPER_PROBE_WATCH", 3*time.Minute)
)

const (
	// Size caps keep the probe folder small enough to send.
	maxCopyFile  = 2 << 20  // one data file
	maxCopyCode  = 12 << 20 // one code file (home booklet jar, page bundle)
	maxCopyTotal = 60 << 20
	maxInventory = 60000 // file entries listed
)

// interesting matches file and folder names worth a copy.
var interesting = regexp.MustCompile(`(?i)home|merch|recommend|store|widget|carousel|discover|storefront|wishlist|card|shelf|collection|cover|thumbnail|readnow|reading.?list|kaf|campaign|feed|banner|dyn`)

// secret matches names that may hold account or device secrets: never
// copied (the probe folder is meant to be shared).
var secret = regexp.MustCompile(`(?i)key|token|cert|cred|cookie|passw|secret|identity|auth|account|\.pem|\.p12|dsn|serial`)

// codeName matches code that likely draws the home screen.
var codeName = regexp.MustCompile(`(?i)home|booklet|launcher|library|storefront|mesquite|pillow|readnow|discover`)

type probe struct {
	mu     sync.Mutex
	out    *os.File
	copied int64
}

func main() {
	if err := os.MkdirAll(filepath.Join(OutDir, "files"), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := os.Create(filepath.Join(OutDir, "report.txt"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	p := &probe{out: f}
	ctx := context.Background()
	start := time.Now()

	_ = screen.Alert("Endpaper probe", "Collecting (about 1 min). Then a 3-minute test: wait for the next box.", 10000)
	p.section("probe")
	p.line("started %s", start.Format(time.RFC3339))

	p.device(ctx)
	p.processes()
	p.appreg(ctx)
	p.lipc(ctx)
	p.inventory()
	p.databases(ctx)
	p.copyRelated()

	_ = screen.Alert("Endpaper probe: please do this now",
		"For 3 minutes: press Home, scroll the whole home page, tap a store/recommended book and go back, "+
			"open the Library, go Home again. Then wait for the 'done' box.", 20000)
	p.watch(ctx)

	p.section("end")
	p.line("finished %s (%s)", time.Now().Format(time.RFC3339), time.Since(start).Round(time.Second))
	_ = screen.Alert("Endpaper probe: done", "Connect USB and send the folder endpaper-probe (zip it).", 15000)
}

// ---- output helpers ----

func (p *probe) section(title string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.out, "\n===== %s =====\n", title)
	p.out.Sync()
}

func (p *probe) line(format string, a ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.out, format+"\n", a...)
}

func (p *probe) block(title, body string) {
	p.section(title)
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintln(p.out, strings.TrimRight(body, "\n"))
	p.out.Sync()
}

func run(ctx context.Context, d time.Duration, name string, args ...string) string {
	c, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	b, err := exec.CommandContext(c, name, args...).CombinedOutput()
	s := string(b)
	if err != nil {
		s += fmt.Sprintf("\n(%s: %v)", name, err)
	}
	return s
}

func readFile(path string, max int) string {
	f, err := os.Open(path)
	if err != nil {
		return "(" + err.Error() + ")"
	}
	defer f.Close()
	b, _ := io.ReadAll(io.LimitReader(f, int64(max)))
	return string(b)
}

// ---- phase 1 ----

func (p *probe) device(ctx context.Context) {
	p.block("firmware", readFile("/etc/prettyversion.txt", 4096))
	p.block("version.txt", readFile("/etc/version.txt", 4096))
	p.block("uname", run(ctx, 5*time.Second, "uname", "-a"))
	p.block("cpuinfo", readFile("/proc/cpuinfo", 8192))
	p.block("meminfo", readFile("/proc/meminfo", 2048))
	p.block("mounts", readFile("/proc/mounts", 16384))
	p.block("df", run(ctx, 10*time.Second, "df", "-k"))
	p.block("eips -i", run(ctx, 5*time.Second, "eips", "-i"))
	p.block("cmdline (kernel)", readFile("/proc/cmdline", 4096))
}

func (p *probe) processes() {
	var b strings.Builder
	dirs, _ := os.ReadDir("/proc")
	type proc struct {
		pid            int
		ppid, rss, exe string
		cmd            string
	}
	var list []proc
	for _, d := range dirs {
		var pid int
		if _, err := fmt.Sscanf(d.Name(), "%d", &pid); err != nil {
			continue
		}
		base := "/proc/" + d.Name()
		cmd := strings.ReplaceAll(strings.TrimRight(readFile(base+"/cmdline", 4096), "\x00"), "\x00", " ")
		exe, _ := os.Readlink(base + "/exe")
		var ppid, rss string
		for _, l := range strings.Split(readFile(base+"/status", 4096), "\n") {
			if strings.HasPrefix(l, "PPid:") {
				ppid = strings.TrimSpace(l[5:])
			}
			if strings.HasPrefix(l, "VmRSS:") {
				rss = strings.TrimSpace(l[6:])
			}
		}
		list = append(list, proc{pid, ppid, rss, exe, cmd})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].pid < list[j].pid })
	for _, x := range list {
		fmt.Fprintf(&b, "%6d ppid %-6s rss %-10s exe %s\n        %s\n", x.pid, x.ppid, x.rss, x.exe, x.cmd)
	}
	p.block("processes", b.String())
}

func (p *probe) appreg(ctx context.Context) {
	p.section("appreg.db (all tables)")
	p.dumpDB(ctx, "/var/local/appreg.db", 5000)
}

// dumpDB writes the schema and up to maxRows rows of every table, read-only.
func (p *probe) dumpDB(ctx context.Context, path string, maxRows int) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(3000)")
	if err != nil {
		p.line("open %s: %v", path, err)
		return
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	rows, err := db.QueryContext(ctx, `SELECT name, sql FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		p.line("%s: %v", path, err)
		return
	}
	type tbl struct{ name, sql string }
	var tables []tbl
	for rows.Next() {
		var n string
		var s sql.NullString
		if rows.Scan(&n, &s) == nil {
			tables = append(tables, tbl{n, s.String})
		}
	}
	rows.Close()
	for _, t := range tables {
		p.line("-- %s\n%s", t.name, t.sql)
		if maxRows <= 0 {
			continue
		}
		r, err := db.QueryContext(ctx, `SELECT * FROM "`+strings.ReplaceAll(t.name, `"`, `""`)+`" LIMIT ?`, maxRows)
		if err != nil {
			p.line("   (%v)", err)
			continue
		}
		cols, _ := r.Columns()
		p.line("   columns: %s", strings.Join(cols, " | "))
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		for r.Next() {
			if r.Scan(ptrs...) != nil {
				continue
			}
			parts := make([]string, len(cols))
			for i, v := range vals {
				s := fmt.Sprint(v)
				if b, ok := v.([]byte); ok {
					s = string(b)
				}
				if len(s) > 300 {
					s = s[:300] + "…"
				}
				parts[i] = s
			}
			p.line("   %s", strings.Join(parts, " | "))
		}
		r.Close()
	}
}

func (p *probe) lipc(ctx context.Context) {
	// Every LIPC service and property name. Without -v: values can hold
	// account or device data.
	p.block("lipc-probe -a", run(ctx, 90*time.Second, "lipc-probe", "-a"))
	var b strings.Builder
	for _, q := range [][2]string{
		{"com.lab126.booklet.home", "getDisplayOptions"},
		{"com.lab126.appmgrd", "activeApp"},
		{"com.lab126.winmgr", "getActiveAppTitle"},
		{"com.lab126.winmgr", "activeDialogCount"},
		{"com.lab126.pillow", "status"},
		{"com.lab126.powerd", "status"},
		{"com.lab126.cmd", "wirelessEnable"},
		{"com.lab126.wifid", "cmState"},
		{"com.lab126.merchant", "status"},
		{"com.lab126.readnow", "status"},
	} {
		fmt.Fprintf(&b, "%s %s = %s\n", q[0], q[1],
			strings.TrimSpace(run(ctx, 5*time.Second, "lipc-get-prop", "-s", q[0], q[1])))
	}
	p.block("lipc values", b.String())
}

// ---- phase 2 ----

var inventoryRoots = []string{
	"/opt/amazon", "/usr/share/webkit-1.0", "/opt/var/local", "/var/local",
	"/mnt/us/system", "/app", "/usr/local", "/etc/upstart", "/usr/java/lib",
	"/opt/ar", "/mnt/base-us/system",
}

func (p *probe) inventory() {
	p.section("file inventory (path, size, mtime)")
	n := 0
	for _, root := range inventoryRoots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				p.line("! %s: %v", path, err)
				return nil
			}
			if n >= maxInventory {
				return filepath.SkipAll
			}
			if d.IsDir() && path == "/var/local/endpaper" {
				return filepath.SkipDir
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			n++
			kind := ""
			if d.IsDir() {
				kind = "/"
			} else if d.Type()&fs.ModeSymlink != 0 {
				t, _ := os.Readlink(path)
				kind = " -> " + t
			}
			p.line("%s%s\t%d\t%s", path, kind, info.Size(), info.ModTime().UTC().Format("2006-01-02 15:04"))
			return nil
		})
	}
	p.line("(%d entries)", n)
}

func (p *probe) databases(ctx context.Context) {
	p.section("sqlite databases (schema only)")
	for _, root := range []string{"/var/local", "/mnt/us/system", "/opt/var/local"} {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || path == "/var/local/appreg.db" {
				return nil
			}
			if !strings.HasSuffix(path, ".db") && !strings.HasSuffix(path, ".sqlite") {
				return nil
			}
			if strings.Contains(path, "/endpaper") {
				return nil
			}
			p.line("\n## %s", path)
			p.dumpDB(ctx, path, 0)
			return nil
		})
	}
}

func (p *probe) copyRelated() {
	p.section("copied files")
	copyOne := func(path string, max int64) {
		if secret.MatchString(path) {
			p.line("skip (private) %s", path)
			return
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > max || p.copied+info.Size() > maxCopyTotal {
			if err == nil && info.Mode().IsRegular() {
				p.line("skip (size %d) %s", info.Size(), path)
			}
			return
		}
		dst := filepath.Join(OutDir, "files", strings.TrimPrefix(path, "/"))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return
		}
		in, err := os.Open(path)
		if err != nil {
			p.line("! %s: %v", path, err)
			return
		}
		defer in.Close()
		out, err := os.Create(dst)
		if err != nil {
			return
		}
		n, _ := io.Copy(out, in)
		out.Close()
		p.copied += n
		p.line("%d\t%s", n, path)
	}
	// Data that may feed the home screen.
	for _, root := range []string{"/var/local", "/mnt/us/system", "/opt/var/local"} {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || strings.Contains(path, "/endpaper") {
				return nil
			}
			if strings.HasSuffix(path, ".db") || strings.Contains(path, "appreg") {
				return nil // databases: schema above; never copy user data DBs
			}
			if interesting.MatchString(path) {
				copyOne(path, maxCopyFile)
			}
			return nil
		})
	}
	// Code that draws the home screen (booklets, web apps, pillow).
	for _, root := range []string{"/opt/amazon/ebook/booklet", "/opt/amazon/ebook/lib", "/usr/share/webkit-1.0/pillow",
		"/opt/var/local/mesquite", "/var/local/mesquite", "/app"} {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if codeName.MatchString(path) && !strings.Contains(path, "/endpaper") {
				copyOne(path, maxCopyCode)
			}
			return nil
		})
	}
	p.line("(%d bytes copied)", p.copied)
}

// ---- phase 3 ----

var watchRoots = []string{"/var/local", "/mnt/us/system", "/opt/var/local", "/tmp"}

func (p *probe) watch(ctx context.Context) {
	p.section(fmt.Sprintf("watch for %s: file changes and app in front", watchFor))
	ctx, cancel := context.WithTimeout(ctx, watchFor)
	defer cancel()
	t0 := time.Now()
	stamp := func() string { return fmt.Sprintf("+%6.1fs", time.Since(t0).Seconds()) }

	// inotify on every folder under the roots (one watch per folder).
	n := 0
	for _, root := range watchRoots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() || n > 4000 || strings.Contains(path, "/endpaper") || strings.HasPrefix(path, "/proc") {
				return nil
			}
			dir := path
			if _, err := events.WatchDir(ctx, dir, func(name string, mask uint32) {
				p.line("%s FILE %s %s", stamp(), maskName(mask), filepath.Join(dir, name))
			}); err == nil {
				n++
			}
			return nil
		})
	}
	p.line("watching %d folders", n)

	// LIPC events of the services that matter for the home screen.
	for _, svc := range []string{"com.lab126.appmgrd", "com.lab126.booklet.home", "com.lab126.pillow",
		"com.lab126.winmgr", "com.lab126.merchant", "com.lab126.readnow", "com.lab126.chromebar"} {
		svc := svc
		_ = events.LIPC(ctx, svc, func(l string) { p.line("%s LIPC %s: %s", stamp(), svc, l) },
			func(f string, a ...any) { p.line("%s lipc %s", stamp(), fmt.Sprintf(f, a...)) })
	}

	// The window in front, every 2 s (only changes are written).
	last := ""
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			now := strings.TrimSpace(run(ctx, 3*time.Second, "lipc-get-prop", "-s", "com.lab126.winmgr", "getActiveAppTitle")) +
				" | app " + strings.TrimSpace(run(ctx, 3*time.Second, "lipc-get-prop", "-s", "com.lab126.appmgrd", "activeApp"))
			if now != last {
				p.line("%s FRONT %s", stamp(), now)
				last = now
			}
		}
	}
}

func maskName(m uint32) string {
	var s []string
	for _, x := range []struct {
		bit  uint32
		name string
	}{{events.InModify, "modify"}, {events.InCloseWrite, "close-write"}, {events.InCreate, "create"},
		{events.InDelete, "delete"}, {events.InMovedTo, "moved-to"}} {
		if m&x.bit != 0 {
			s = append(s, x.name)
		}
	}
	return strings.Join(s, ",")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func durOr(k string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(k)); err == nil {
		return d
	}
	return def
}
