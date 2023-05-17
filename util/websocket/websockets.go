package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dchest/uniuri"
	"github.com/gorilla/websocket"
	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
	backoff2 "github.com/kubecube-io/kubecube-e2e/util/retry"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"net/http"
	"sync"
)

type WebSocket struct {
	sync.Mutex
	Address          string
	TransportAddress string
	ServerID         string
	SessionID        string
	Connection       *websocket.Conn
	Inbound          chan []byte
	Reconnected      chan struct{}
	Stop             chan struct{}
}

func NewWebSocket(address string, sessionId string, stop chan struct{}) (*WebSocket, error) {
	ws := &WebSocket{
		Address:     address,
		ServerID:    paddedRandomIntn(999),
		SessionID:   uniuri.NewLen(8),
		Inbound:     make(chan []byte),
		Reconnected: make(chan struct{}, 32),
		Stop:        stop,
	}

	ws.TransportAddress = address + "/" + ws.ServerID + "/" + ws.SessionID + "/websocket"

	ws.Loop(sessionId)

	return ws, nil
}

func (w *WebSocket) Loop(sessionId string) {
	go func() {
		b := &backoff2.ExponentialBackOff{
			InitialInterval:     framework.WaitInterval,
			RandomizationFactor: backoff2.DefaultRandomizationFactor,
			Multiplier:          backoff2.DefaultMultiplier,
			MaxInterval:         backoff2.DefaultMaxInterval,
			MaxElapsedTime:      framework.WaitTimeout,
			Clock:               backoff2.SystemClock,
		}
		b.Reset()
		connectFunc := func() error {
			clog.Info("Starting a WebSocket connection to %s", w.TransportAddress)
			ws, _, err := websocket.DefaultDialer.Dial(w.TransportAddress, http.Header{})
			if err != nil {
				clog.Info(err.Error())
				return err
			}

			// Read the open message
			_, data, err := ws.ReadMessage()
			if err != nil {
				return err
			}
			if data[0] != 'o' {
				err := errors.New("invalid initial message")
				return err
			}

			message, err := GetBindData(sessionId).GetWriteMessage()
			if err != nil {
				return err
			}
			err = ws.WriteJSON([]string{message})
			if err != nil {
				return err
			}
			w.Connection = ws
			w.Reconnected <- struct{}{}

			for {
				_, data, err := w.Connection.ReadMessage()
				if err != nil {
					return err
				}

				if len(data) < 1 {
					continue
				}

				switch data[0] {
				case 'h':
					// Heartbeat
					clog.Info("get health websocket")
					continue
				case 'a':
					// Normal message
					clog.Info("get normal message: %s", string(data))
					w.Inbound <- data[1:]
				case 'c':
					// Session closed
					var v []interface{}
					if err := json.Unmarshal(data[1:], &v); err != nil {
						clog.Error("Closing session: %s", err)
						return nil
					}
					break
				default:
					clog.Info("get unknown websocket message: %s", string(data))
					continue
				}
			}
		}
		err := backoff2.Retry(connectFunc, b, context.Background())
		if err != nil {
			clog.Error(err.Error())
		}
	}()

	<-w.Reconnected
}

func (w *WebSocket) ReadJSON(v interface{}) error {
	message := <-w.Inbound
	return json.Unmarshal(message, v)
}

func (w *WebSocket) WriteJSON(v interface{}) error {
	return w.Connection.WriteJSON(v)
}

func (w *WebSocket) Close() error {
	<-w.Stop
	return w.Connection.Close()
}
