package aslince

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Aslince struct {
	tb.Bot
	redis       *redis.Pool
	lastMessage tb.Message
	paintChance int
}

func NewAslince(r *redis.Pool, b tb.Bot) *Aslince {
	return &Aslince{
		redis:       r,
		Bot:         b,
		paintChance: 5,
	}
}

var sources = map[string][]string{
	"зе флоу":       {"the-flow", "The Flow"},
	"тасс":          {"tass", "tass.ru", "ТАСС"},
	"двач":          {"2ch", "двач"},
	"мдк":           {"mdk", "мдк"},
	"рифмы и панчи": {"рифмы и панчи"},
}

func msgLogger(u *tb.Update) bool {
	if u.Message != nil {
		log.Debugf("got message '%s' from %d:%s", textFromMsg(u.Message), u.Message.Sender.ID, u.Message.Sender.FirstName)
	}
	return true
}

func chatFilter(u *tb.Update) bool {
	if u.Message == nil || u.Message.Chat == nil {
		return true
	}
	if u.Message.Chat.Title != "твитор ОПГ" && u.Message.Chat.Title != "predlozhka_test_chat" {
		log.Debugf("skip message '%s' from %d:%s", textFromMsg(u.Message), u.Message.Sender.ID, u.Message.Sender.FirstName)
		return false
	}
	return true
}

func (a *Aslince) Start() {
	a.Handle(tb.OnText, a.handle)
	a.Handle(tb.OnPhoto, a.handle)
	a.Handle(tb.OnVideo, a.handle)

	poller := tb.NewMiddlewarePoller(a.Poller, msgLogger)
	poller = tb.NewMiddlewarePoller(poller, chatFilter)

	a.Poller = poller
	a.Bot.Start()
	a.startBackgroundJobs()
}

func (a *Aslince) getStatus() (string, error) {
	conn := a.redis.Get()
	defer conn.Close()
	b := strings.Builder{}
	for s := range sources {
		count, err := redis.Int(conn.Do("GET", dailySrcKey(s)))
		if err != nil && err != redis.ErrNil {
			return "", err
		}
		if count > 0 {
			b.WriteString(fmt.Sprintf("☑️ %s\n", s))
		} else {
			b.WriteString(fmt.Sprintf("❌ %s\n", s))
		}
	}
	return b.String(), nil
}

func chance(c int) bool {
	return rand.Intn(99)+1 <= c
}

func (a *Aslince) handle(m *tb.Message) {
	if isComand(m.Text) {
		if strings.Contains(m.Text, "че там") || strings.Contains(m.Text, "чо там") || strings.Contains(m.Text, "чё там") || strings.Contains(m.Text, "че сегодня") {
			status, err := a.getStatus()
			if err != nil {
				log.Error(err)
				a.Send(m.Chat, "да нихуя", &tb.SendOptions{ReplyTo: m})
			}
			a.Send(m.Chat, status, &tb.SendOptions{ReplyTo: m})
		}

		if strings.Contains(m.Text, "рисуй меньше") {
			if a.paintChance > 10 {
				a.paintChance -= 10
				a.Send(m.Chat, "да бля", &tb.SendOptions{ReplyTo: m})
			} else {
				a.Send(m.Chat, "затравили", &tb.SendOptions{ReplyTo: m})
			}
		}

		if strings.Contains(m.Text, "рисуй больше") {
			if a.paintChance <= 90 {
				a.paintChance += 10
				a.Send(m.Chat, "ладно", &tb.SendOptions{ReplyTo: m})
			} else {
				a.Send(m.Chat, "больше не могу(", &tb.SendOptions{ReplyTo: m})
			}
		}

		if strings.Contains(m.Text, "не рисуй") {
			a.paintChance = 0
			a.Send(m.Chat, "травля((", &tb.SendOptions{ReplyTo: m})
		}
	}

	if m.Photo != nil && chance(a.paintChance) {
		photo, err := a.paint(m)
		if err != nil {
			log.Error("can't paint", err)
		}
		_, err = a.Send(m.Chat, photo)
		if err != nil {
			log.Error(err)
		}
	}

	for name, checks := range sources {
		if checkLinks(m, checks) {
			err := a.replySuccessCheck(m, name)
			if err != nil {
				log.Error("error reply success check link:", err)
			}
		}
		if m.IsForwarded() && m.OriginalChat != nil {
			if checkText(m.OriginalChat.Title, checks) {
				err := a.replySuccessCheck(m, name)
				if err != nil {
					log.Error("error reply success check forward:", err)
				}
			}
		}
	}

}

func checkLinks(m *tb.Message, checks []string) bool {
	for _, e := range m.Entities {
		var text string
		if e.Type == tb.EntityURL {
			text = textFromMsg(m)[e.Offset : e.Offset+e.Length]
		} else if e.Type != tb.EntityTextLink {
			text = e.URL
		}
		if checkText(text, checks) {
			return true
		}
	}
	return false
}

func checkText(text string, checks []string) bool {
	for _, c := range checks {
		if strings.Contains(strings.ToLower(text), strings.ToLower(c)) {
			return true
		}
	}
	return false
}

var namespace = "aslince"

func TimeIn(t time.Time, name string) (time.Time, error) {
	loc, err := time.LoadLocation(name)
	if err == nil {
		t = t.In(loc)
	}
	return t, err
}

func dailySrcKey(src string) string {
	t, err := TimeIn(time.Now(), "Europe/Moscow")
	if err != nil {
		t = time.Now()
	}
	return fmt.Sprintf("%s:%s:%s", namespace, src, t.Format("02-01-2006"))
}

func (a *Aslince) replySuccessCheck(m *tb.Message, source string) error {
	conn := a.redis.Get()
	defer conn.Close()

	key := dailySrcKey(source)
	_, err := conn.Do("SET", key, 0, "NX", "EX", time.Duration(time.Hour*24).Seconds())
	if err != nil {
		log.Error("set err ", err)
		return err
	}
	count, err := redis.Int(conn.Do("INCR", key))
	if err != nil {
		log.Error("inc err ", err)
		return err
	}

	if count > 1 {
		return nil
	}

	_, err = a.Send(
		m.Chat,
		fmt.Sprintf("☑️ %s – чек", source), &tb.SendOptions{
			ReplyTo: m,
		})
	if err != nil {
		return err
	}

	return nil
}

func textFromMsg(m *tb.Message) string {
	text := m.Text
	if text == "" {
		text = m.Caption
	}
	return text
}

func isComand(text string) bool {
	t := strings.ToLower(text)
	return strings.Contains(t, "аслица") || strings.Contains(t, "аслинце") || strings.Contains(t, "ослица") || strings.Contains(t, "@Aslincevtelege")
}
