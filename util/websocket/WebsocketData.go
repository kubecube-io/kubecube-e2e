package websocket

import "encoding/json"

const (
	BindOp  = "bind"
	StdinOp = "stdin"
)

type Data struct {
	Op        string `json:"Op"`
	SessionID string `json:"SessionID,omitempty"`
	Data      string `json:"Data,omitempty"`
}

func GetBindData(sessionId string) *Data {
	return &Data{
		Op:        BindOp,
		SessionID: sessionId,
	}
}

func GetOpData(op string) *Data {
	return &Data{
		Op:   StdinOp,
		Data: op,
	}
}

func (data *Data) GetWriteMessage() (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
