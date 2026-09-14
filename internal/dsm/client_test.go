package dsm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/laurenschristian/dsmctl/internal/dsm"
	"github.com/laurenschristian/dsmctl/internal/dsm/dsmtest"
)

func newClient(t *testing.T) (*dsm.Client, *dsmtest.Server) {
	t.Helper()
	srv := dsmtest.New("me", "pw")
	c := dsm.New(srv.URL)
	t.Cleanup(srv.Close)
	return c, srv
}

func TestLoginSuccess(t *testing.T) {
	c, _ := newClient(t)
	sess, err := c.Login(context.Background(), "me", "pw", "")
	if err != nil {
		t.Fatal(err)
	}
	if sess.SID == "" || c.SID == "" {
		t.Fatal("no sid set")
	}
	if sess.SynoToken == "" {
		t.Fatal("no synotoken")
	}
}

func TestLoginBadPassword(t *testing.T) {
	c, _ := newClient(t)
	_, err := c.Login(context.Background(), "me", "wrong", "")
	var apiErr *dsm.APIError
	if !asAPI(err, &apiErr) || apiErr.Code != 400 {
		t.Fatalf("want code 400, got %v", err)
	}
}

func TestLoginOTPRequired(t *testing.T) {
	srv := dsmtest.New("me", "pw")
	srv.OTP = "654321"
	defer srv.Close()
	c := dsm.New(srv.URL)
	if _, err := c.Login(context.Background(), "me", "pw", ""); err == nil {
		t.Fatal("expected 403 without otp")
	}
	if _, err := c.Login(context.Background(), "me", "pw", "654321"); err != nil {
		t.Fatalf("otp login failed: %v", err)
	}
}

func TestSessionRequired(t *testing.T) {
	c, _ := newClient(t)
	// No login: server rejects with 119.
	if _, err := c.System(context.Background()); err == nil {
		t.Fatal("expected session error")
	}
}

func TestReadEndpoints(t *testing.T) {
	c, _ := newClient(t)
	if _, err := c.Login(context.Background(), "me", "pw", ""); err != nil {
		t.Fatal(err)
	}
	sys, err := c.System(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sys.Model != "DS1821+" || sys.RAMSize != 16384 {
		t.Fatalf("bad system info: %+v", sys)
	}
	if _, err := c.Utilization(context.Background()); err != nil {
		t.Fatal(err)
	}
	st, err := c.Storage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Volumes) != 1 || len(st.Disks) != 1 || st.Disks[0].SMARTStatus != "normal" {
		t.Fatalf("bad storage: %+v", st)
	}
	ctx := context.Background()
	for _, get := range []func(context.Context) (json.RawMessage, error){
		c.Packages, c.DockerContainers, c.Tasks, c.Shares, c.Users, c.Connections,
	} {
		if _, err := get(ctx); err != nil {
			t.Fatal(err)
		}
	}
}

func TestErrorText(t *testing.T) {
	e := &dsm.APIError{API: "SYNO.API.Auth", Code: 403}
	if got := e.Error(); got == "" {
		t.Fatal("empty error text")
	}
	if !(&dsm.APIError{Code: 106}).SessionExpired() {
		t.Fatal("106 should be session-expired")
	}
	if (&dsm.APIError{Code: 400}).SessionExpired() {
		t.Fatal("400 is not session-expired")
	}
}

// helpers

func asAPI(err error, target **dsm.APIError) bool {
	for err != nil {
		if e, ok := err.(*dsm.APIError); ok {
			*target = e
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func TestErrorTextCodes(t *testing.T) {
	cases := []struct {
		api  string
		code int
	}{
		{"SYNO.API.Auth", 401}, {"SYNO.API.Auth", 402}, {"SYNO.API.Auth", 404},
		{"SYNO.API.Auth", 406}, {"SYNO.API.Auth", 407}, {"SYNO.API.Auth", 408},
		{"SYNO.Core.System", 100}, {"SYNO.Core.System", 101}, {"SYNO.Core.System", 102},
		{"SYNO.Core.System", 103}, {"SYNO.Core.System", 104}, {"SYNO.Core.System", 105},
		{"SYNO.Core.System", 107},
	}
	for _, c := range cases {
		if msg := (&dsm.APIError{API: c.api, Code: c.code}).Error(); msg == "" {
			t.Fatalf("no text for %s/%d", c.api, c.code)
		}
	}
	// Unknown code still yields a non-empty error string.
	if (&dsm.APIError{API: "X", Code: 9999}).Error() == "" {
		t.Fatal("empty error for unknown code")
	}
}

func TestHTTPErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()
	c := dsm.New(srv.URL)
	if err := c.Ping(context.Background()); err == nil {
		t.Fatal("expected HTTP 500 error")
	}
}

func TestNewTrimsSlash(t *testing.T) {
	if c := dsm.New("http://h:5000/"); c.BaseURL != "http://h:5000" {
		t.Fatalf("slash not trimmed: %q", c.BaseURL)
	}
}
