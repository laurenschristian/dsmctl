package dsm

import (
	"context"
	"encoding/json"
	"net/url"
)

// SystemInfo is the SYNO.Core.System info payload (fields are best-effort; DSM
// omits some by model).
type SystemInfo struct {
	Model          string `json:"model"`
	Serial         string `json:"serial"`
	FirmwareVer    string `json:"firmware_ver"`
	Version        string `json:"version_string"`
	RAMSize        int    `json:"ram_size"`
	Uptime         string `json:"up_time"`
	Temperature    int    `json:"sys_temp"`
	TempWarn       bool   `json:"temperature_warning"`
	CPUSeries      string `json:"cpu_series"`
	CPUCores       string `json:"cpu_cores"`
	NTPEnabled     bool   `json:"ntp_enable"`
	SysTempSupport bool   `json:"sys_tempwarn"`
	TimeZone       string `json:"time_zone"`
	SystemTime     string `json:"time"`
}

func (c *Client) System(ctx context.Context) (*SystemInfo, error) {
	var out SystemInfo
	if err := c.Call(ctx, "SYNO.Core.System", "info", 3, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Utilization is a subset of SYNO.Core.System.Utilization get.
type Utilization struct {
	CPU struct {
		UserLoad   json.Number `json:"user_load"`
		SystemLoad json.Number `json:"system_load"`
		OtherLoad  json.Number `json:"other_load"`
	} `json:"cpu"`
	Memory struct {
		RealUsage json.Number `json:"real_usage"`
		TotalReal json.Number `json:"total_real"`
		Cached    json.Number `json:"cached"`
	} `json:"memory"`
}

func (c *Client) Utilization(ctx context.Context) (*Utilization, error) {
	var out Utilization
	if err := c.Call(ctx, "SYNO.Core.System.Utilization", "get", 1, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Volume is one storage volume from SYNO.Storage.CGI.Storage load_info.
type Volume struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	FSType string `json:"fs_type"`
	Size   struct {
		Total json.Number `json:"total"`
		Used  json.Number `json:"used"`
	} `json:"size"`
}

// Disk is one physical disk (with SMART) from load_info.
type Disk struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Model       string `json:"model"`
	Vendor      string `json:"vendor"`
	Serial      string `json:"serial"`
	Status      string `json:"status"`
	SMARTStatus string `json:"smart_status"`
	Temp        int    `json:"temp"`
	SizeTotal   string `json:"size_total"`
	Container   string `json:"container_type"`
	Unc         int    `json:"unc"`
}

type StorageInfo struct {
	Volumes []Volume `json:"volumes,omitempty"`
	Disks   []Disk   `json:"disks,omitempty"`
}

func (c *Client) Storage(ctx context.Context) (*StorageInfo, error) {
	var out StorageInfo
	if err := c.Call(ctx, "SYNO.Storage.CGI.Storage", "load_info", 1, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// The rest return raw payloads: DSM field shapes vary by version and are shown
// as pretty JSON. `dsmctl raw` reaches anything not covered here.

func (c *Client) Packages(ctx context.Context) (json.RawMessage, error) {
	p := url.Values{}
	p.Set("additional", `["status"]`)
	return c.Raw(ctx, "SYNO.Core.Package", "list", 2, p)
}

func (c *Client) DockerContainers(ctx context.Context) (json.RawMessage, error) {
	p := url.Values{}
	p.Set("limit", "-1")
	p.Set("offset", "0")
	p.Set("type", "all")
	return c.Raw(ctx, "SYNO.Docker.Container", "list", 1, p)
}

func (c *Client) Tasks(ctx context.Context) (json.RawMessage, error) {
	p := url.Values{}
	p.Set("offset", "0")
	p.Set("limit", "-1")
	return c.Raw(ctx, "SYNO.Core.TaskScheduler", "list", 1, p)
}

func (c *Client) Shares(ctx context.Context) (json.RawMessage, error) {
	p := url.Values{}
	p.Set("additional", `["hidden","encryption","is_aclmode","desc","disable_list","disable_modify","disable_download"]`)
	return c.Raw(ctx, "SYNO.Core.Share", "list", 1, p)
}

func (c *Client) Users(ctx context.Context) (json.RawMessage, error) {
	p := url.Values{}
	p.Set("type", "local")
	p.Set("offset", "0")
	p.Set("limit", "-1")
	p.Set("additional", `["email","description","expired"]`)
	return c.Raw(ctx, "SYNO.Core.User", "list", 1, p)
}

func (c *Client) Connections(ctx context.Context) (json.RawMessage, error) {
	return c.Raw(ctx, "SYNO.Core.CurrentConnection", "list", 1, nil)
}

// Reboot restarts the DSM host. Gate this behind a confirmation in the CLI.
func (c *Client) Reboot(ctx context.Context) error {
	return c.Call(ctx, "SYNO.Core.System", "reboot", 2, nil, nil)
}

// Shutdown powers off the DSM host. Gate this behind a confirmation in the CLI.
func (c *Client) Shutdown(ctx context.Context) error {
	return c.Call(ctx, "SYNO.Core.System", "shutdown", 2, nil, nil)
}

// UpdateInfo is the SYNO.Core.Upgrade.Server check payload.
type UpdateInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Reboot    string `json:"reboot"`
	Restart   string `json:"restart"`
	Type      string `json:"type"`
}

func (c *Client) UpdateCheck(ctx context.Context) (*UpdateInfo, error) {
	var out UpdateInfo
	if err := c.Call(ctx, "SYNO.Core.Upgrade.Server", "check", 1, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Security(ctx context.Context) (json.RawMessage, error) {
	return c.Raw(ctx, "SYNO.Core.SecurityScan.Status", "system_get", 1, nil)
}

func (c *Client) Network(ctx context.Context) (json.RawMessage, error) {
	return c.Raw(ctx, "SYNO.DSM.Network", "list", 2, nil)
}

func (c *Client) UPS(ctx context.Context) (json.RawMessage, error) {
	return c.Raw(ctx, "SYNO.Core.ExternalDevice.UPS", "get", 1, nil)
}

func (c *Client) Processes(ctx context.Context) (json.RawMessage, error) {
	return c.Raw(ctx, "SYNO.Core.System.Process", "list", 1, nil)
}

func (c *Client) Groups(ctx context.Context) (json.RawMessage, error) {
	return c.Raw(ctx, "SYNO.Core.Group", "list", 1, nil)
}

func (c *Client) BackupTasks(ctx context.Context) (json.RawMessage, error) {
	return c.Raw(ctx, "SYNO.Backup.Task", "list", 1, nil)
}

// APIs lists every WebAPI the host exposes (name to version/path). Unauthenticated.
func (c *Client) APIs(ctx context.Context) (json.RawMessage, error) {
	p := url.Values{}
	p.Set("query", "all")
	return c.Raw(ctx, "SYNO.API.Info", "query", 1, p)
}

// UPSInfo is a subset of SYNO.Core.ExternalDevice.UPS get.
type UPSInfo struct {
	Enable      bool        `json:"enable"`
	Status      string      `json:"status"`
	Charge      json.Number `json:"charge"`
	Runtime     json.Number `json:"runtime"`
	Model       string      `json:"model"`
	Manufacture string      `json:"manufacture"`
	Mode        string      `json:"mode"`
}

func (c *Client) UPSInfo(ctx context.Context) (*UPSInfo, error) {
	var out UPSInfo
	if err := c.Call(ctx, "SYNO.Core.ExternalDevice.UPS", "get", 1, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
