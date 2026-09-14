package dsmtest_test

import (
	"context"
	"testing"

	"github.com/laurenschristian/dsmctl/internal/dsm"
	"github.com/laurenschristian/dsmctl/internal/dsm/dsmtest"
)

// TestServerSmoke exercises every branch of the fake server so its coverage is
// attributed to this package.
func TestServerSmoke(t *testing.T) {
	srv := dsmtest.New("me", "pw")
	defer srv.Close()
	c := dsm.New(srv.URL)
	ctx := context.Background()

	if err := c.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Login(ctx, "me", "pw", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.Logout(ctx); err != nil {
		t.Fatal(err)
	}
	// Re-login for authed reads.
	if _, err := c.Login(ctx, "me", "pw", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := c.System(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Utilization(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Storage(ctx); err != nil {
		t.Fatal(err)
	}
	for _, get := range []func(context.Context) ([]byte, error){
		asBytes(c.Packages), asBytes(c.DockerContainers), asBytes(c.Tasks),
		asBytes(c.Shares), asBytes(c.Users), asBytes(c.Connections),
	} {
		if _, err := get(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Reboot(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	// Unknown API returns 102.
	if _, err := c.Raw(ctx, "SYNO.Nope", "x", 1, nil); err == nil {
		t.Fatal("expected error for unknown api")
	}
}

func asBytes[T ~[]byte](f func(context.Context) (T, error)) func(context.Context) ([]byte, error) {
	return func(ctx context.Context) ([]byte, error) {
		v, err := f(ctx)
		return []byte(v), err
	}
}
