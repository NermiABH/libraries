package inform

import (
	"fmt"
	"github.com/NermiABH/libraries/web"
	"log"
	"time"
)

type Sender struct {
	in       chan string
	done     chan struct{}
	token    string
	chatID   string
	threadId string
	web      *web.Web
}

func NewSender(subsystem, token, chatID, threadId string, queue int) *Sender {
	s := &Sender{
		in:       make(chan string, queue),
		done:     make(chan struct{}),
		token:    token,
		chatID:   chatID,
		threadId: threadId,
		web:      web.New(subsystem + "-telegram"),
	}
	go s.daemon()
	return s
}

func (s *Sender) ToQueue(text string) {
	s.in <- text
}

func (s *Sender) Stop() {
	close(s.in)
	<-s.done
}

func (s *Sender) Send(text string) error {
	url := "https://api.telegram.org/bot" + s.token + "/sendMessage"
	body := "chat_id=" + s.chatID + "&message_thread_id=" + s.threadId + "&text=" + text
	_, code, err := s.web.Post(url,
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		[]byte(body),
		time.Second*10)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("code not 200 (%d)", code)
	}
	return nil
}

func (s *Sender) daemon() {
	for text := range s.in {
		err := s.Send(text)
		if err != nil {
			log.Printf("[TG FAIL] %v", err)
		}
	}
	close(s.done)
}
