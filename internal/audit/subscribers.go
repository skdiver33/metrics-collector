// Package audit.
// Модуль определяет возможных получаетелей сообщений аудита.
// Текущая реализация включает ведене аудита в файл - FileSubscriber,
// Отправка сообщений аудита на сервер - URLSubscriber.
package audit

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

const (
	FileSubscriberType = "file"
	URLSubscriberType  = "url"
)

type FileSubscriber struct {
	fileName       string
	subscriberType string
	mu             *sync.Mutex
}

func NewFileSubscriber(filename string) *FileSubscriber {

	return &FileSubscriber{fileName: filename, subscriberType: FileSubscriberType, mu: &sync.Mutex{}}
}

func (fs *FileSubscriber) getType() string {
	return fs.subscriberType
}

func (fs *FileSubscriber) update(msg []byte) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	file, err := os.OpenFile(fs.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Print("Audit error! Error open audit file")
		return
	}
	defer file.Close()

	_, err = file.Write(msg)
	if err != nil {
		log.Printf("Audit error! Error write in audit file %s", err.Error())
		return
	}
	fmt.Fprintln(file)

}

type URLSubscriber struct {
	url           string
	subscribeType string
}

func NewURLSubscriber(url string) *URLSubscriber {
	return &URLSubscriber{url: url, subscribeType: URLSubscriberType}
}

func (us *URLSubscriber) getType() string {
	return us.subscribeType
}

func (us *URLSubscriber) update(msg []byte) {
	tr := &http.Transport{}
	client := &http.Client{Transport: tr}
	var requestBody bytes.Buffer
	requestBody.Write(msg)
	req, err := http.NewRequest(http.MethodPost, us.url, &requestBody)
	if err != nil {
		log.Print("Audit error! Error create request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		log.Println("Error send POST to audit service")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		log.Println("Status not OK")
	}
}
