package dsm

import (
	"context"
	"errors"
	"net/url"
)

// AuthVersion is the SYNO.API.Auth version used for login. DSM 7 supports 7.
const AuthVersion = 7

// Session holds a logged-in DSM session token and CSRF token.
type Session struct {
	SID       string `json:"sid"`
	SynoToken string `json:"synotoken,omitempty"`
}

// Login authenticates and stores the session on the client. otp may be empty
// when the account has no 2-factor authentication.
func (c *Client) Login(ctx context.Context, account, passwd, otp string) (Session, error) {
	p := url.Values{}
	p.Set("account", account)
	p.Set("passwd", passwd)
	p.Set("format", "sid")
	p.Set("enable_syno_token", "yes")
	if otp != "" {
		p.Set("otp_code", otp)
	}
	var out Session
	if err := c.Call(ctx, "SYNO.API.Auth", "login", AuthVersion, p, &out); err != nil {
		return Session{}, err
	}
	if out.SID == "" {
		return Session{}, errors.New("login returned no session id")
	}
	return out, c.setSession(out.SID, out.SynoToken)
}

// Logout invalidates the current session on the DSM host.
func (c *Client) Logout(ctx context.Context) error {
	if c.SID == "" {
		return nil
	}
	return c.Call(ctx, "SYNO.API.Auth", "logout", AuthVersion, nil, nil)
}

func (c *Client) setSession(sid, synoToken string) error {
	c.SID = sid
	c.SynoToken = synoToken
	if c.OnSession != nil {
		return c.OnSession(sid, synoToken)
	}
	return nil
}

// Ping checks reachability with an unauthenticated API.Info query.
func (c *Client) Ping(ctx context.Context) error {
	p := url.Values{}
	p.Set("query", "SYNO.API.Auth")
	return c.Call(ctx, "SYNO.API.Info", "query", 1, p, nil)
}
