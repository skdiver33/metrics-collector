// Модуль позволяет выполнять аудит обновления(добавления) метрик.
// Сообщение аудита имеет формат JSON и может сохраняться в указанном файле, отправляться на указанный сервер.
package audit

import (
	"encoding/json"
	"time"
)

type AuditMessage struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IP        string   `json:"ip_address"`
}

type AuditPublisher interface {
	Register(AuditObserver)
	Deregister(AuditObserver)
	notify()
}

type AuditObserver interface {
	update([]byte)
	getType() string
}

type AuditEvent struct {
	observers   map[string]AuditObserver
	jsonMessage []byte
}

func NewAuditEvent() *AuditEvent {
	return &AuditEvent{}
}

func (e *AuditEvent) Register(o AuditObserver) {
	if e.observers == nil {
		e.observers = make(map[string]AuditObserver)
	}
	e.observers[o.getType()] = o
}

func (e *AuditEvent) Deregister(o AuditObserver) {
	delete(e.observers, o.getType())
}

func (e *AuditEvent) notify() {
	for _, observer := range e.observers {
		observer.update(e.jsonMessage)
	}
}

func (e *AuditEvent) Update(metrics []string, ip string) error {
	msg := AuditMessage{}
	msg.Timestamp = time.Now().Unix()
	msg.IP = ip
	msg.Metrics = metrics
	message, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	e.jsonMessage = message
	e.notify()
	return nil
}
