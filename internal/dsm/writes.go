package dsm

import (
	"context"
	"encoding/json"
	"net/url"
)

// DockerAction runs start, stop, or restart on a container by name.
func (c *Client) DockerAction(ctx context.Context, action, name string) error {
	p := url.Values{}
	p.Set("name", name)
	return c.Call(ctx, "SYNO.Docker.Container", action, 1, p, nil)
}

// RunTask triggers a scheduled task by id (as shown by `dsmctl tasks`).
func (c *Client) RunTask(ctx context.Context, id string) error {
	p := url.Values{}
	p.Set("id", id)
	return c.Call(ctx, "SYNO.Core.TaskScheduler", "run", 1, p, nil)
}

// PackageAction runs start or stop on an installed package by id. DSM wants the
// id as a JSON array.
func (c *Client) PackageAction(ctx context.Context, action, id string) error {
	ids, err := json.Marshal([]string{id})
	if err != nil {
		return err
	}
	p := url.Values{}
	p.Set("id", string(ids))
	return c.Call(ctx, "SYNO.Core.Package.Control", action, 1, p, nil)
}
