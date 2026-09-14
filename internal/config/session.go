package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Session is the cached DSM login token, kept apart from config.yaml so a stale
// token never rides along in a committed or shared config.
type Session struct {
	SID       string `json:"sid"`
	SynoToken string `json:"synotoken,omitempty"`
}

func SessionPath() string {
	return filepath.Join(filepath.Dir(Path()), "session.json")
}

func LoadSession() Session {
	var s Session
	if b, err := os.ReadFile(SessionPath()); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func SaveSession(s Session) error {
	p := SessionPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}

func ClearSession() error {
	err := os.Remove(SessionPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
