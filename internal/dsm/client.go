// Package dsm is a thin client for Synology DiskStation Manager (DSM 7 WebAPI).
//
// Every request goes to /webapi/entry.cgi with api/version/method params and an
// optional _sid session token. Responses use the envelope
// {"success":bool,"data":{...},"error":{"code":N}}.
package dsm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to a DSM host. SID and SynoToken come from Login; OnSession is
// called whenever they change so the caller can persist the session.
type Client struct {
	BaseURL   string
	SID       string
	SynoToken string
	OnSession func(sid, synoToken string) error
	HTTP      *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// ErrNotLoggedIn is returned when a request needs a session but none is set.
var ErrNotLoggedIn = errors.New("not logged in: run `dsmctl login`")

// APIError is a DSM error-code response.
type APIError struct {
	API  string
	Code int
}

func (e *APIError) Error() string {
	if msg := errText(e.API, e.Code); msg != "" {
		return fmt.Sprintf("dsm %s error %d: %s", e.API, e.Code, msg)
	}
	return fmt.Sprintf("dsm %s error %d", e.API, e.Code)
}

// SessionExpired reports whether the code means the SID is stale and a re-login
// would help.
func (e *APIError) SessionExpired() bool {
	switch e.Code {
	case 105, 106, 107, 119:
		return true
	}
	return false
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code int `json:"code"`
	} `json:"error"`
}

// Call posts one request to entry.cgi and unmarshals data into out (may be nil).
func (c *Client) Call(ctx context.Context, api, method string, version int, params url.Values, out any) error {
	raw, err := c.call(ctx, api, method, version, params)
	if err != nil {
		return err
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// Raw returns the data payload of any endpoint for `dsmctl raw` and MCP.
func (c *Client) Raw(ctx context.Context, api, method string, version int, params url.Values) (json.RawMessage, error) {
	return c.call(ctx, api, method, version, params)
}

func (c *Client) call(ctx context.Context, api, method string, version int, params url.Values) (json.RawMessage, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("api", api)
	params.Set("method", method)
	params.Set("version", fmt.Sprintf("%d", version))
	if c.SID != "" {
		params.Set("_sid", c.SID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/webapi/entry.cgi", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	// DSM binds an enable_syno_token session to this CSRF header; a request
	// without it is rejected as "session id not found" (119), reads included.
	if c.SynoToken != "" {
		req.Header.Set("X-SYNO-TOKEN", c.SynoToken)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var env envelope
	if err := json.Unmarshal(b, &env); err != nil {
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("%s.%s: HTTP %d: %s", api, method, resp.StatusCode, strings.TrimSpace(string(b)))
		}
		return nil, fmt.Errorf("%s.%s: bad response: %w", api, method, err)
	}
	if env.Success {
		return env.Data, nil
	}
	code := 0
	if env.Error != nil {
		code = env.Error.Code
	}
	return nil, &APIError{API: api, Code: code}
}

// errText maps the common documented DSM error codes to a message.
func errText(api string, code int) string {
	if api == "SYNO.API.Auth" {
		switch code {
		case 400:
			return "no such account or incorrect password"
		case 401:
			return "account disabled"
		case 402:
			return "permission denied"
		case 403:
			return "2-factor authentication code required (pass --otp)"
		case 404:
			return "2-factor authentication code failed"
		case 406:
			return "2-factor authentication enforced (pass --otp)"
		case 407:
			return "IP address blocked"
		case 408, 409, 410:
			return "password expired or must be changed"
		}
	}
	switch code {
	case 100:
		return "unknown error"
	case 101:
		return "invalid parameter"
	case 102:
		return "requested API does not exist"
	case 103:
		return "requested method does not exist"
	case 104:
		return "requested version not supported"
	case 105:
		return "insufficient permission (session may be stale)"
	case 106:
		return "session timeout"
	case 107:
		return "session interrupted by duplicate login"
	case 119:
		return "session id not found (run `dsmctl login`)"
	}
	return ""
}
