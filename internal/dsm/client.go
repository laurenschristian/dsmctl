// Package dsm is a thin client for Synology DiskStation Manager (DSM 7 WebAPI).
package dsm

import (
	"net/http"
	"strings"
	"time"
)

// Client talks to the dsmctl target. Endpoints land here as commands are built.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &http.Client{Timeout: 30 * time.Second}}
}
