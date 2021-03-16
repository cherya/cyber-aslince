package aslince

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Aslince struct {
	tb.Bot
	redis       *redis.Pool
	lastMessage *tb.Message
	paintChance int
	talk        *Talker
}

func NewAslince(r *redis.Pool, b tb.Bot) *Aslince {
	a := &Aslince{
		redis:       r,
		Bot:         b,
		paintChance: 5,
	}
	chatExport, err := ioutil.ReadFile("opg/result.json")
	if err != nil {
		log.Warn("chat export not found")
		return a
	}
	model, err := ioutil.ReadFile("model.json")
	if err != nil {
		log.Warn("model.json not found")
		model = []byte{}
	}
	t, err := NewTalker(chatExport, model)
	if err != nil {
		return a
	}
	a.talk = t

	return a
}

var sources = map[string][]string{
	"зе флоу":       {"the-flow", "The Flow"},
	"тасс":          {"tass", "tass.ru", "ТАСС"},
	"двач":          {"2ch", "двач"},
	"мдк":           {"mdk", "мдк"},
	"рифмы и панчи": {"рифмы и панчи"},
	"сбер":          {},
}

func msgLogger(u *tb.Update) bool {
	if u.Message != nil {
		log.Debugf("got message '%s' from %d:%s in chat %d", textFromMsg(u.Message), u.Message.Sender.ID, u.Message.Sender.FirstName, u.Message.Chat.ID)
		if u.Message.Voice != nil {
			log.Debug(u.Message.Voice)
		}
	}
	return true
}

func chatFilter(u *tb.Update) bool {
	// 	if u.Message == nil {
	// 		return true
	// 	}
	// 	if u.Message.Chat == nil || u.Message.Private() {
	// 		return true
	// 	}
	// 	if u.Message.Chat.Title != "твитор ОПГ" && u.Message.Chat.Title != "predlozhka_test_chat" && u.Message.Chat.ID != -1001169505246 {
	// 		log.Debugf("skip message '%s' from %d:%s", textFromMsg(u.Message), u.Message.Sender.ID, u.Message.Sender.FirstName)
	// 		return false
	// 	}
	return true
}

func (a *Aslince) Start() {
	a.Handle(tb.OnText, a.handle)
	a.Handle(tb.OnPhoto, a.handle)
	a.Handle(tb.OnVideo, a.handle)

	poller := tb.NewMiddlewarePoller(a.Poller, msgLogger)
	poller = tb.NewMiddlewarePoller(poller, chatFilter)

	a.Poller = poller
	a.startBackgroundJobs()
	a.Bot.Start()
}

func (a *Aslince) Shutdown() error {
	a.Bot.Stop()
	model := a.talk.GetModel()
	if model != nil {
		log.Debug("saving model...")
		err := saveModel(model)
		if err != nil {
			return err
		}
		log.Debug("model saved")
		return nil
	}
	log.Debug("model is empty, nothing to save")
	return nil
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

func (a *Aslince) handleCommand(text string, m *tb.Message) {
	if m.ReplyTo != nil && m.ReplyTo.Photo != nil {
		photo, err := a.paint(m.ReplyTo)
		if err != nil {
			log.Error("can't paint", err)
		}
		_, err = a.Send(m.Chat, photo)
		if err != nil {
			log.Error(err)
		}
		return
	}

	if m.Photo != nil {
		photo, err := a.paint(m)
		if err != nil {
			log.Error("can't paint", err)
		}
		_, err = a.Send(m.Chat, photo)
		if err != nil {
			log.Error(err)
		}
		return
	}

	if chetamRegex.MatchString(text) {
		status, err := a.getStatus()
		if err != nil {
			log.Error(err)
			a.Send(m.Chat, "да нихуя", &tb.SendOptions{ReplyTo: m})
		}
		a.Send(m.Chat, status, &tb.SendOptions{ReplyTo: m})
		return
	}

	if strings.Contains(text, "рисуй меньше") {
		if a.paintChance > 10 {
			a.paintChance -= 10
			a.Send(m.Chat, "да бля", &tb.SendOptions{ReplyTo: m})
		} else {
			a.Send(m.Chat, "затравили", &tb.SendOptions{ReplyTo: m})
		}
		return
	}

	if strings.Contains(text, "рисуй больше") {
		if a.paintChance <= 90 {
			a.paintChance += 10
			a.Send(m.Chat, "ладно", &tb.SendOptions{ReplyTo: m})
		} else {
			a.Send(m.Chat, "больше не могу(", &tb.SendOptions{ReplyTo: m})
		}
		return
	}

	if strings.Contains(text, "не рисуй") {
		a.paintChance = 0
		a.Send(m.Chat, "травля((", &tb.SendOptions{ReplyTo: m})
		return
	}

	if strings.Contains(text, "шансы") {
		a.Send(m.Chat, fmt.Sprintf("%d/100", a.paintChance), &tb.SendOptions{ReplyTo: m})
		return
	}

	err := a.answer(m)
	if err != nil {
		log.Error("handleCommand: ", err)
	}
}

func (a *Aslince) answer(m *tb.Message) error {
	if a.talk != nil {
		text := a.talk.GenerateMessage(m.Text)
		_, err := a.Send(m.Chat, text, &tb.SendOptions{ReplyTo: m})
		if err != nil {
			return errors.Wrapf(err, "can't answer to message %s", m.ID)
		}
	}
	return nil
}

var chetamRegex = regexp.MustCompile("ч([еёо]|(то)) (там|сегодня)")

func (a *Aslince) handle(m *tb.Message) {
	a.lastMessage = m
	text := strings.ToLower(m.Text)
	if isComand(text) {
		a.handleCommand(text, m)
	} else if m.IsReply() && m.ReplyTo.Sender.ID == a.Me.ID || m.Private() {
		err := a.answer(m)
		if err != nil {
			log.Error("handle: ", err)
		}
	} else {
		a.talk.Add(m.Text)
	}

	if m.Photo != nil && chance(a.paintChance) || m.Private() && m.Photo != nil {
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
	if m.Sender.ID == 418166693 {
		err := a.replySuccessCheck(m, "тасс")
		if err != nil {
			log.Error("error reply success check Vitalya:", err)
		}
	}
	if m.Sender.ID == 95123848 {
		err := a.replySuccessCheck(m, "сбер")
		if err != nil {
			log.Error("error reply success check sber:", err)
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

func (a *Aslince) checkBingo() bool {
	conn := a.redis.Get()
	defer conn.Close()

	for s := range sources {
		val, err := redis.Int(conn.Do("GET", dailySrcKey(s)))
		if err != nil || val < 1 {
			return false
		}
	}
	return true
}

func (a *Aslince) replySuccessCheck(m *tb.Message, source string) error {
	conn := a.redis.Get()
	defer conn.Close()

	key := dailySrcKey(source)
	_, err := conn.Do("SET", key, 0, "NX", "EX", (time.Hour * 24).Seconds())
	if err != nil {
		log.Error("set err ", err)
		return err
	}
	count, err := redis.Int(conn.Do("INCR", key))
	if err != nil {
		log.Error("inc err ", err)
		return err
	}

	bingo := a.checkBingo()
	if bingo {
		set, err := redis.String(conn.Do("SET", dailySrcKey("bingo"), 0, "EX", (time.Hour * 24).Seconds()))
		if err != nil {
			return err
		}
		if set != "OK" {
			log.Info("bingo++")
			return nil
		}
		_, err = a.Send(
			m.Chat,
			&tb.Voice{File: tb.File{FileID: "AwACAgIAAxkBAAILRF-lgaG47OvQjYWQnrqVu7oT_Ft8AALgAwAC1JiQSDQf3N0GBXlMHgQ"}},
			&tb.SendOptions{
				ReplyTo: m,
			},
		)
		if err != nil {
			return err
		}
		return nil
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

var aslinceRegexp = regexp.MustCompile("(([ао]сли)+(ца|нце)|(@Aslincevtelege))")

func isComand(text string) bool {
	return aslinceRegexp.MatchString(strings.ToLower(text))
}
