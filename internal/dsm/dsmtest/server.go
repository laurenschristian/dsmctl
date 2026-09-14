// Package dsmtest is an in-memory fake DSM WebAPI server for tests.
package dsmtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

// Server is a fake DSM host. It validates login and hands out a session id, then
// answers a fixed set of entry.cgi APIs.
type Server struct {
	*httptest.Server
	Account  string
	Password string
	OTP      string // if set, login without a matching otp_code fails with 403
	SID      string
	Requests int
}

// New starts a fake DSM server. Close it when done.
func New(account, password string) *Server {
	s := &Server{Account: account, Password: password, SID: "sid-abc123"}
	mux := http.NewServeMux()
	mux.HandleFunc("/webapi/entry.cgi", s.entry)
	s.Server = httptest.NewServer(mux)
	return s
}

func fail(w http.ResponseWriter, code int) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   map[string]int{"code": code},
	})
}

func ok(w http.ResponseWriter, data any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
}

func (s *Server) entry(w http.ResponseWriter, r *http.Request) {
	s.Requests++
	_ = r.ParseForm()
	api := r.FormValue("api")
	method := r.FormValue("method")
	w.Header().Set("Content-Type", "application/json")

	// Session check for everything except Auth and API.Info.
	if api != "SYNO.API.Auth" && api != "SYNO.API.Info" {
		if r.FormValue("_sid") != s.SID {
			fail(w, 119) // session id not found
			return
		}
	}

	switch api {
	case "SYNO.API.Info":
		ok(w, map[string]any{
			"SYNO.API.Auth":    map[string]any{"minVersion": 1, "maxVersion": 7, "path": "entry.cgi"},
			"SYNO.Core.System": map[string]any{"minVersion": 1, "maxVersion": 3, "path": "entry.cgi"},
		})
	case "SYNO.API.Auth":
		s.auth(w, r, method)
	case "SYNO.Core.System":
		s.system(w, method)
	case "SYNO.Core.System.Utilization":
		ok(w, map[string]any{
			"cpu":    map[string]any{"user_load": "5", "system_load": "2", "other_load": "1"},
			"memory": map[string]any{"real_usage": "34", "total_real": "16384", "cached": "2048"},
		})
	case "SYNO.Storage.CGI.Storage":
		ok(w, map[string]any{
			"volumes": []any{map[string]any{"id": "volume_1", "status": "normal", "fs_type": "btrfs",
				"size": map[string]any{"total": "1000", "used": "400"}}},
			"disks": []any{map[string]any{"id": "sata1", "name": "Drive 1", "model": "WD40", "status": "normal",
				"smart_status": "normal", "temp": 34, "size_total": "4000787030016"}},
		})
	case "SYNO.Core.Package":
		ok(w, map[string]any{"packages": []any{map[string]any{"id": "Docker", "name": "Container Manager", "version": "24.0"}}})
	case "SYNO.Docker.Container":
		ok(w, map[string]any{"containers": []any{map[string]any{"name": "adguard", "image": "adguard/adguardhome", "status": "running"}}})
	case "SYNO.Core.TaskScheduler":
		ok(w, map[string]any{"tasks": []any{map[string]any{"id": 1, "name": "Backup", "enable": true}}})
	case "SYNO.Core.Share":
		ok(w, map[string]any{"shares": []any{map[string]any{"name": "docker", "vol_path": "/volume1"}}})
	case "SYNO.Core.User":
		ok(w, map[string]any{"users": []any{map[string]any{"name": "admin", "email": "a@b.c"}}})
	case "SYNO.Core.CurrentConnection":
		ok(w, map[string]any{"items": []any{map[string]any{"who": "admin", "type": "SMB"}}})
	case "SYNO.Core.Upgrade.Server":
		ok(w, map[string]any{"available": true, "version": "DSM 7.4.1-90080", "reboot": "now", "restart": "none", "type": "dsm"})
	case "SYNO.Core.SecurityScan.Status":
		ok(w, map[string]any{"items": map[string]any{"malware": map[string]any{"failSeverity": "safe", "progress": 100}}})
	case "SYNO.DSM.Network":
		ok(w, map[string]any{"dns": []any{"1.1.1.1"}, "gateway": "10.0.4.1",
			"interfaces": []any{map[string]any{"id": "ovs_eth0", "mac": "90-09-D0-62-EE-35"}}})
	case "SYNO.Core.ExternalDevice.UPS":
		ok(w, map[string]any{"enable": true, "charge": 99, "model": "Back-UPS XS 1500M", "mode": "USB"})
	case "SYNO.Core.System.Process":
		ok(w, map[string]any{"process": []any{map[string]any{"command": "sscamerad", "cpu": 47, "mem": 71896, "pid": 14114}}})
	case "SYNO.Core.Group":
		ok(w, map[string]any{"groups": []any{map[string]any{"name": "administrators", "description": ""}}})
	case "SYNO.Backup.Task":
		ok(w, map[string]any{"task_list": []any{}, "total": 0})
	default:
		fail(w, 102) // no such API
	}
}

func (s *Server) auth(w http.ResponseWriter, r *http.Request, method string) {
	switch method {
	case "login":
		if r.FormValue("account") != s.Account || r.FormValue("passwd") != s.Password {
			fail(w, 400)
			return
		}
		if s.OTP != "" && r.FormValue("otp_code") != s.OTP {
			fail(w, 403)
			return
		}
		data := map[string]any{"sid": s.SID}
		if r.FormValue("enable_syno_token") == "yes" {
			data["synotoken"] = "csrf-xyz"
		}
		ok(w, data)
	case "logout":
		ok(w, map[string]any{})
	default:
		fail(w, 103)
	}
}

func (s *Server) system(w http.ResponseWriter, method string) {
	switch method {
	case "info":
		ok(w, map[string]any{
			"model": "DS1821+", "serial": "SN123", "version_string": "DSM 7.2.1-69057",
			"firmware_ver": "7.2.1", "ram_size": 16384, "up_time": "34:17:36",
			"cpu_series": "AMD Ryzen V1500B", "cpu_cores": "4", "sys_temp": 41, "temperature_warning": false,
		})
	case "reboot", "shutdown":
		ok(w, map[string]any{})
	default:
		fail(w, 103)
	}
}
