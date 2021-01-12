package aslince

import (
	"time"

	log "github.com/sirupsen/logrus"
)

var deadChatMessage = "чят здох"

type ChatRecipient struct {
	id string
}

func (c ChatRecipient) Recipient() string {
	return c.id
}

func (a *Aslince) startBackgroundJobs() {
	log.Info("background jobs started")
	deadChatTicker := time.NewTicker(time.Minute)
	go func() {
		for {
			select {
			case <-deadChatTicker.C:
				if a.deadChatCheck() {
					a.lastMessage, _ = a.Send(ChatRecipient{id: "-1001261346511"}, deadChatMessage)
				}
			}
		}
	}()
}

func (a *Aslince) deadChatCheck() bool {
	if a.lastMessage == nil {
		return false
	}
	mt, err := TimeIn(a.lastMessage.Time(), "Europe/Moscow")
	if err != nil {
		mt = a.lastMessage.Time()
	}
	t, err := TimeIn(time.Now(), "Europe/Moscow")
	if err != nil {
		t = time.Now()
	}
	if t.Sub(mt).Hours() > 3 && a.lastMessage.Text != deadChatMessage {
		return true
	}
	return false
}
