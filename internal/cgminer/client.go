// Package cgminer implements a minimal client for the CGMiner-style API
// exposed by BOSminer (TCP, JSON request, NUL-terminated JSON reply).
package cgminer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

// Client queries a BOSminer API endpoint.
type Client struct {
	Address string
	Timeout time.Duration
}

// Status is one entry of the STATUS array present in every reply.
type Status struct {
	Status      string `json:"STATUS"`
	When        int64  `json:"When"`
	Code        int    `json:"Code"`
	Msg         string `json:"Msg"`
	Description string `json:"Description"`
}

// Response is a decoded reply: the STATUS array plus the raw
// command-specific arrays keyed by name (e.g. "FANS", "TEMPS").
type Response struct {
	ID     int
	Status []Status
	Data   map[string][]map[string]any
}

type request struct {
	Command   string `json:"command"`
	Parameter any    `json:"parameter,omitempty"`
}

// Query sends a single read-only command and decodes the reply.
func (c *Client) Query(ctx context.Context, command string) (*Response, error) {
	raw, err := c.roundTrip(ctx, request{Command: command})
	if err != nil {
		return nil, err
	}
	return decode(raw)
}

func (c *Client) roundTrip(ctx context.Context, req request) ([]byte, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.Address)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", c.Address, err)
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write(payload); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	raw, err := io.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return bytes.TrimRight(raw, "\x00\n\r "), nil
}

func decode(raw []byte) (*Response, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("decode reply: %w", err)
	}
	resp := &Response{Data: map[string][]map[string]any{}}
	for k, v := range m {
		switch k {
		case "id":
			_ = json.Unmarshal(v, &resp.ID)
		case "STATUS":
			if err := json.Unmarshal(v, &resp.Status); err != nil {
				return nil, fmt.Errorf("decode STATUS: %w", err)
			}
		default:
			var rows []map[string]any
			if err := json.Unmarshal(v, &rows); err == nil {
				resp.Data[k] = rows
			}
		}
	}
	if len(resp.Status) > 0 && resp.Status[0].Status == "E" {
		return resp, fmt.Errorf("miner error: %s", resp.Status[0].Msg)
	}
	return resp, nil
}

// Float extracts a numeric field from a row (JSON numbers decode as float64).
func Float(row map[string]any, key string) (float64, bool) {
	f, ok := row[key].(float64)
	return f, ok
}

// String extracts a string field from a row.
func String(row map[string]any, key string) (string, bool) {
	s, ok := row[key].(string)
	return s, ok
}
