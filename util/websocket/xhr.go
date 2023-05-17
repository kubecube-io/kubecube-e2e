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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/dchest/uniuri"
	"github.com/igm/sockjs-go/sockjs"
	"sync"
)

type XHR struct {
	Address          string
	TransportAddress string
	ServerID         string
	SessionID        string
	Inbound          chan []byte
	Done             chan bool
	sessionState     sockjs.SessionState
	mu               sync.RWMutex
}

var client = http.Client{Timeout: time.Second * 10}

func NewXHR(address string) (*XHR, error) {
	xhr := &XHR{
		Address:      address,
		ServerID:     paddedRandomIntn(999),
		SessionID:    uniuri.NewLen(8),
		Inbound:      make(chan []byte),
		Done:         make(chan bool, 1),
		sessionState: sockjs.SessionOpening,
	}
	xhr.TransportAddress = address + "/" + xhr.ServerID + "/" + xhr.SessionID
	if err := xhr.Init(); err != nil {
		return nil, err
	}
	go xhr.StartReading()

	return xhr, nil
}

func (x *XHR) Init() error {
	req, err := http.NewRequest("POST", x.TransportAddress+"/xhr", nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if body[0] != 'o' {
		return errors.New("Invalid initial message")
	}
	x.setSessionState(sockjs.SessionActive)

	return nil
}

func (x *XHR) doneNotify() {
	return
}

func (x *XHR) StartReading() {
	client := &http.Client{Timeout: time.Second * 30}
	for {
		select {
		case <-x.Done:
			return
		default:
			req, err := http.NewRequest("POST", x.TransportAddress+"/xhr", nil)
			if err != nil {
				clog.Error(err.Error())
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				clog.Error(err.Error())
				continue
			}

			data, err := ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				clog.Error(err.Error())
				continue
			}

			switch data[0] {
			case 'h':
				// Heartbeat
				continue
			case 'a':
				// Normal message
				x.Inbound <- data[1:]
			case 'c':
				// Session closed
				x.setSessionState(sockjs.SessionClosed)
				var v []interface{}
				if err := json.Unmarshal(data[1:], &v); err != nil {
					clog.Error("Closing session: %s", err)
					break
				}
				clog.Info("%v: %v", v[0], v[1])
				break
			default:
				clog.Info("get unknown websocket message: %s", string(data))
				continue
			}
		}
	}
}

func (x *XHR) ReadJSON(v interface{}) error {
	message := <-x.Inbound
	return json.Unmarshal(message, v)
}

func (x *XHR) WriteJSON(v interface{}) error {
	message, err := json.Marshal(v)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", x.TransportAddress+"/xhr_send", bytes.NewReader(message))
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return errors.New("Invalid HTTP code - " + resp.Status)
	}

	return nil
}

func (x *XHR) Close() error {
	select {
	case x.Done <- true:
	default:
		return fmt.Errorf("Error closing XHR")
	}
	return nil
}

func (x *XHR) GetSessionState() sockjs.SessionState {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.sessionState
}

func (x *XHR) setSessionState(state sockjs.SessionState) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.sessionState = state
}
