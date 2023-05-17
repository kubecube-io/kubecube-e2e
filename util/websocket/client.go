/*
Copyright 2023 KubeCube Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package websocket

import (
	"encoding/json"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	Connection Connection

	WebSockets   bool
	Address      string
	ReadBufSize  int
	WriteBufSize int

	Reconnected chan struct{}
}

func NewClient(address string, sessionId string, stop chan struct{}) (*Client, error) {
	client := &Client{}

	client.Address = address

	// Get info whether WebSockets are enabled
	info, err := client.Info()
	if err != nil {
		return nil, err
	}
	client.WebSockets = info.WebSocket

	// Create a WS session (not a SJS one)
	if client.WebSockets {
		a2 := strings.Replace(address, "https", "wss", 1)
		a2 = strings.Replace(a2, "http", "ws", 1)

		ws, err := NewWebSocket(a2, sessionId, stop)
		if err != nil {
			return nil, err
		}

		client.Connection = ws
		client.Reconnected = ws.Reconnected
	} else {
		// XHR
		client.Connection, err = NewXHR(address)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (c *Client) Info() (*Info, error) {
	resp, err := http.Get(c.Address + "/info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	clog.Info("session msg: %s", body)
	var info Info
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func (c *Client) WriteMessage(p interface{}) error {
	return c.Connection.WriteJSON(p)
}

func (c *Client) ReadMessage(p interface{}) error {
	return c.Connection.ReadJSON(p)
}

func (c *Client) Close() error {
	return c.Connection.Close()
}
