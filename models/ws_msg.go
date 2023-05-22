package models

type WsMsg struct {
	Data      interface{} `json:"data"`
	EventType string      `json:"event_type"`
	Uri       string      `json:"uri"`
}
