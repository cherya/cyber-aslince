package aslince

import "time"

var deadChatMessage = "чят здох"

type ChatRecipient struct {
	id string
}

func (c ChatRecipient) Recipient() string {
	return c.id
}

func (a *Aslince) startBackgroundJobs() {
	deadChatTicker := time.NewTicker(time.Minute)
	go func() {
		for {
			select {
			case <-deadChatTicker.C:
				if a.deadChatCheck() {
					a.Send(ChatRecipient{id: "29462028"}, deadChatMessage)
				}
			}
		}
	}()
}

func (a *Aslince) deadChatCheck() bool {
	mt, err := TimeIn(a.lastMessage.Time(), "Europe/Moscow")
	if err != nil {
		mt = a.lastMessage.Time()
	}
	t, err := TimeIn(time.Now(), "Europe/Moscow")
	if err != nil {
		t = time.Now()
	}
	if t.Hour() > 7 || t.Hour() <= 1 {
		if t.Sub(mt).Hours() > 1 && a.lastMessage.Text != deadChatMessage {
			return true
		}
	}
	return false
}
