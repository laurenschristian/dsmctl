package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/laurenschristian/dsmctl/internal/config"
	"github.com/laurenschristian/dsmctl/internal/dsm"
	"github.com/laurenschristian/dsmctl/internal/dsm/dsmtest"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	flagJSON = false
	root := Root()
	root.SetArgs(args)
	err := root.Execute()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String(), err
}

// fakeEnv points config at a fresh fake DSM server plus an isolated config dir.
func fakeEnv(t *testing.T) *dsmtest.Server {
	t.Helper()
	srv := dsmtest.New("me", "pw")
	t.Cleanup(srv.Close)
	t.Setenv("DSM_CONFIG", t.TempDir()+"/config.yaml")
	t.Setenv("DSM_URL", srv.URL)
	t.Setenv("DSM_USER", "me")
	t.Setenv("DSM_PASS", "pw")
	return srv
}

func TestDoctor(t *testing.T) {
	srv := fakeEnv(t)
	_ = srv
	out, err := run(t, "doctor")
	if err != nil || !strings.Contains(out, "url") {
		t.Fatalf("%v\n%s", err, out)
	}
	if out, _ := run(t, "doctor", "--json"); !strings.Contains(out, "\"reachable\": true") {
		t.Fatalf("json doctor: %s", out)
	}
}

func TestNoURLErrors(t *testing.T) {
	t.Setenv("DSM_CONFIG", os.DevNull)
	t.Setenv("DSM_URL", "")
	if _, err := run(t, "status"); err == nil {
		t.Fatal("expected error without url")
	}
}

func TestStatusAndHealth(t *testing.T) {
	fakeEnv(t)
	out, err := run(t, "status")
	if err != nil || !strings.Contains(out, "DS1821+") {
		t.Fatalf("status %v\n%s", err, out)
	}
	out, err = run(t, "status", "--json")
	if err != nil || !strings.Contains(out, "\"model\"") {
		t.Fatalf("status json %v\n%s", err, out)
	}
	out, err = run(t, "health")
	if err != nil || !strings.Contains(out, "volume") || !strings.Contains(out, "SMART") {
		t.Fatalf("health %v\n%s", err, out)
	}
}

func TestReadCommands(t *testing.T) {
	fakeEnv(t)
	for _, tc := range []struct {
		cmd, want string
	}{
		{"storage", "sata1"},
		{"packages", "Container Manager"},
		{"docker", "adguard"},
		{"tasks", "Backup"},
		{"shares", "docker"},
		{"users", "admin"},
		{"connections", "SMB"},
		{"security", "malware"},
		{"network", "10.0.4.1"},
		{"ups", "Back-UPS"},
		{"processes", "sscamerad"},
		{"groups", "administrators"},
		{"backup", "task_list"},
	} {
		out, err := run(t, tc.cmd)
		if err != nil || !strings.Contains(out, tc.want) {
			t.Fatalf("%s: %v\n%s", tc.cmd, err, out)
		}
	}
}

func TestRawCommand(t *testing.T) {
	fakeEnv(t)
	out, err := run(t, "raw", "SYNO.Core.System", "info", "--version", "3")
	if err != nil || !strings.Contains(out, "DS1821+") {
		t.Fatalf("raw %v\n%s", err, out)
	}
}

func TestLoginLogout(t *testing.T) {
	fakeEnv(t)
	out, err := run(t, "login", "--json")
	if err != nil || !strings.Contains(out, "logged_in") {
		t.Fatalf("login %v\n%s", err, out)
	}
	if config.LoadSession().SID == "" {
		t.Fatal("session not cached")
	}
	if _, err := run(t, "logout"); err != nil {
		t.Fatal(err)
	}
	if config.LoadSession().SID != "" {
		t.Fatal("session not cleared")
	}
}

func TestBadPasswordSurfacesError(t *testing.T) {
	srv := dsmtest.New("me", "pw")
	t.Cleanup(srv.Close)
	t.Setenv("DSM_CONFIG", t.TempDir()+"/config.yaml")
	t.Setenv("DSM_URL", srv.URL)
	t.Setenv("DSM_USER", "me")
	t.Setenv("DSM_PASS", "wrong")
	_, err := run(t, "status")
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("want auth 400, got %v", err)
	}
}

func TestMCPTools(t *testing.T) {
	srv := dsmtest.New("me", "pw")
	t.Cleanup(srv.Close)
	cfg = &config.Config{URL: srv.URL, Username: "me", Password: "pw"}
	client = dsm.New(srv.URL)

	s := mcpServer()
	ct, st := mcp.NewInMemoryTransports()
	go func() { _ = s.Run(context.Background(), st) }()
	sess, err := mcp.NewClient(&mcp.Implementation{Name: "t"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	tools, err := sess.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) < 17 {
		t.Fatalf("want >=17 tools, got %d", len(tools.Tools))
	}
	names := map[string]bool{}
	for _, tl := range tools.Tools {
		names[tl.Name] = true
	}
	for _, want := range []string{"dsm_status", "dsm_health", "dsm_storage", "dsm_docker", "dsm_raw"} {
		if !names[want] {
			t.Fatalf("missing tool %s", want)
		}
	}

	for _, name := range []string{
		"dsm_status", "dsm_health", "dsm_storage", "dsm_update", "dsm_security",
		"dsm_network", "dsm_ups", "dsm_processes", "dsm_packages", "dsm_docker",
		"dsm_tasks", "dsm_backup", "dsm_shares", "dsm_users", "dsm_groups",
		"dsm_connections", "dsm_apis",
	} {
		res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{Name: name})
		if err != nil || res.IsError {
			t.Fatalf("%s: err=%v result=%v", name, err, res)
		}
	}
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "dsm_raw",
		Arguments: map[string]any{"api": "SYNO.Core.System", "method": "info", "version": 3},
	})
	if err != nil || res.IsError {
		t.Fatalf("dsm_raw %v", err)
	}
}

func TestInitWritesConfig(t *testing.T) {
	cfgPath := t.TempDir() + "/config.yaml"
	t.Setenv("DSM_CONFIG", cfgPath)
	t.Setenv("DSM_URL", "")
	t.Setenv("DSM_USER", "")
	t.Setenv("DSM_PASS", "")
	out, err := run(t, "init", "--url", "http://nas:5000", "--user", "me",
		"--password-cmd", "echo pw")
	if err != nil || !strings.Contains(out, "wrote") {
		t.Fatalf("init %v\n%s", err, out)
	}
	c, _ := config.Load()
	if c.URL != "http://nas:5000" || c.Username != "me" || c.PasswordCmd != "echo pw" {
		t.Fatalf("bad config %+v", c)
	}
}

func TestRebootShutdownGate(t *testing.T) {
	fakeEnv(t)
	// Without --yes and no tty input, the command aborts.
	if _, err := run(t, "reboot"); err == nil {
		t.Fatal("reboot without --yes should abort")
	}
	if _, err := run(t, "reboot", "--yes"); err != nil {
		t.Fatalf("reboot --yes: %v", err)
	}
	if _, err := run(t, "shutdown", "--yes"); err != nil {
		t.Fatalf("shutdown --yes: %v", err)
	}
}

func TestDoctorUnreachable(t *testing.T) {
	t.Setenv("DSM_CONFIG", os.DevNull)
	t.Setenv("DSM_URL", "http://127.0.0.1:1")
	t.Setenv("DSM_USER", "")
	t.Setenv("DSM_PASS", "")
	out, err := run(t, "doctor", "--json")
	if err != nil || !strings.Contains(out, "\"reachable\": false") {
		t.Fatalf("doctor unreachable %v\n%s", err, out)
	}
}

func TestSessionExpiredRetry(t *testing.T) {
	srv := dsmtest.New("me", "pw")
	t.Cleanup(srv.Close)
	t.Setenv("DSM_CONFIG", t.TempDir()+"/config.yaml")
	t.Setenv("DSM_URL", srv.URL)
	t.Setenv("DSM_USER", "me")
	t.Setenv("DSM_PASS", "pw")
	// Prime a stale session so the first authed call gets 119, then re-login.
	_ = config.SaveSession(config.Session{SID: "stale"})
	out, err := run(t, "status")
	if err != nil || !strings.Contains(out, "DS1821+") {
		t.Fatalf("retry status %v\n%s", err, out)
	}
}

func TestJSONRawOutput(t *testing.T) {
	fakeEnv(t)
	out, err := run(t, "packages", "--json")
	if err != nil || !strings.Contains(out, "Container Manager") {
		t.Fatalf("packages json %v\n%s", err, out)
	}
	out, err = run(t, "storage", "--json")
	if err != nil || !strings.Contains(out, "sata1") {
		t.Fatalf("storage json %v\n%s", err, out)
	}
}

func TestLoginNoPassword(t *testing.T) {
	srv := dsmtest.New("me", "pw")
	t.Cleanup(srv.Close)
	t.Setenv("DSM_CONFIG", t.TempDir()+"/config.yaml")
	t.Setenv("DSM_URL", srv.URL)
	t.Setenv("DSM_USER", "me")
	t.Setenv("DSM_PASS", "")
	if _, err := run(t, "login"); err == nil {
		t.Fatal("expected no-password error")
	}
}

func TestNotLoggedInWithoutCreds(t *testing.T) {
	srv := dsmtest.New("me", "pw")
	t.Cleanup(srv.Close)
	t.Setenv("DSM_CONFIG", os.DevNull)
	t.Setenv("DSM_URL", srv.URL)
	t.Setenv("DSM_USER", "")
	t.Setenv("DSM_PASS", "")
	_, err := run(t, "status")
	if err == nil || !strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("want not-logged-in, got %v", err)
	}
}

func TestHelpers(t *testing.T) {
	if firstNonEmpty("", "b", "c") != "b" {
		t.Fatal("firstNonEmpty")
	}
	if firstNonEmpty("", "") != "" {
		t.Fatal("firstNonEmpty empty")
	}
	if uptime(&dsm.SystemInfo{Uptime: "10:00:00"}) != "10:00:00" {
		t.Fatal("uptime passthrough")
	}
	if uptime(&dsm.SystemInfo{}) != "unknown" {
		t.Fatal("uptime fallback")
	}
	if !confirm(true, "x") {
		t.Fatal("confirm --yes")
	}
}

func TestUpdateAndApis(t *testing.T) {
	fakeEnv(t)
	out, err := run(t, "update")
	if err != nil || !strings.Contains(out, "DSM 7.4.1-90080") {
		t.Fatalf("update %v\n%s", err, out)
	}
	out, err = run(t, "apis")
	if err != nil || !strings.Contains(out, "SYNO.Core.System") {
		t.Fatalf("apis %v\n%s", err, out)
	}
	out, err = run(t, "apis", "auth")
	if err != nil || !strings.Contains(out, "SYNO.API.Auth") {
		t.Fatalf("apis filter %v\n%s", err, out)
	}
}

func TestLoginPositionalOTP(t *testing.T) {
	srv := dsmtest.New("me", "pw")
	srv.OTP = "424242"
	t.Cleanup(srv.Close)
	t.Setenv("DSM_CONFIG", t.TempDir()+"/config.yaml")
	t.Setenv("DSM_URL", srv.URL)
	t.Setenv("DSM_USER", "me")
	t.Setenv("DSM_PASS", "pw")
	out, err := run(t, "login", "424242")
	if err != nil || !strings.Contains(out, "session cached") {
		t.Fatalf("positional otp login %v\n%s", err, out)
	}
}

func TestNeedsOTP(t *testing.T) {
	if !needsOTP(&dsm.APIError{API: "SYNO.API.Auth", Code: 403}) {
		t.Fatal("403 should need otp")
	}
	if needsOTP(&dsm.APIError{API: "SYNO.API.Auth", Code: 400}) {
		t.Fatal("400 is not an otp prompt")
	}
	if needsOTP(&dsm.APIError{API: "SYNO.Core.System", Code: 403}) {
		t.Fatal("non-auth 403 is not an otp prompt")
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{0: "0 B", 512: "512 B", 1024: "1.0 KiB", 1048576: "1.0 MiB"}
	for n, want := range cases {
		if got := humanI(n); got != want {
			t.Fatalf("humanI(%d)=%q want %q", n, got, want)
		}
	}
	if pct(68, 100) != "68%" || pct(1, 0) != "n/a" {
		t.Fatal("pct")
	}
}

func TestDockerActions(t *testing.T) {
	fakeEnv(t)
	// gated: no --yes with no tty input aborts
	if _, err := run(t, "docker", "restart", "adguard"); err == nil {
		t.Fatal("restart without --yes should abort")
	}
	for _, action := range []string{"start", "stop", "restart"} {
		out, err := run(t, "docker", action, "adguard", "--yes")
		if err != nil || !strings.Contains(out, action) {
			t.Fatalf("docker %s: %v\n%s", action, err, out)
		}
	}
}

func TestTaskRun(t *testing.T) {
	fakeEnv(t)
	if _, err := run(t, "tasks", "run", "12"); err == nil {
		t.Fatal("task run without --yes should abort")
	}
	out, err := run(t, "tasks", "run", "12", "--yes")
	if err != nil || !strings.Contains(out, "started") {
		t.Fatalf("task run: %v\n%s", err, out)
	}
}

func TestPackageActions(t *testing.T) {
	fakeEnv(t)
	for _, action := range []string{"start", "stop"} {
		out, err := run(t, "packages", action, "Docker", "--yes")
		if err != nil || !strings.Contains(out, action) {
			t.Fatalf("package %s: %v\n%s", action, err, out)
		}
	}
}

func TestDockerActionMethods(t *testing.T) {
	srv := dsmtest.New("me", "pw")
	t.Cleanup(srv.Close)
	c := dsm.New(srv.URL)
	if _, err := c.Login(context.Background(), "me", "pw", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.DockerAction(context.Background(), "restart", "x"); err != nil {
		t.Fatal(err)
	}
	if err := c.RunTask(context.Background(), "1"); err != nil {
		t.Fatal(err)
	}
	if err := c.PackageAction(context.Background(), "start", "Docker"); err != nil {
		t.Fatal(err)
	}
}
