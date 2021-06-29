package aslince

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strings"

	"github.com/cherya/cyber-aslince/internal/daily_plan"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Aslince struct {
	tb.Bot
	redis       *redis.Pool
	lastMessage *tb.Message
	paintChance int
	talk        *Talker
	plan        *daily_plan.DailyPlan
}

var redisNamespace = "aslince"
var chetamRegex = regexp.MustCompile("ч([еёо]|(то)) (там|сегодня)")
var eeeeeBoiRegex = regexp.MustCompile("[eе]* бо[йи]")
var aslinceRegexp = regexp.MustCompile("(([ао]сли)+(нце)|(@Aslincevtelege))")

func NewAslince(r *redis.Pool, b tb.Bot) *Aslince {
	a := &Aslince{
		redis:       r,
		Bot:         b,
		paintChance: 5,
		plan:        daily_plan.New(r, redisNamespace),
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

func msgLogger(u *tb.Update) bool {
	//if u.Message != nil {
	//	log.Debugf("got message '%s' from %d:%s in chat %d", u.Message.Sender.ID, u.Message.Sender.FirstName, u.Message.Chat.ID)
	//	if u.Message.Voice != nil {
	//		log.Debug(u.Message.Voice)
	//	}
	//}
	return true
}

func (a *Aslince) Start() {
	a.Handle(tb.OnText, a.handle)
	a.Handle(tb.OnPhoto, a.handle)
	a.Handle(tb.OnVideo, a.handle)

	poller := tb.NewMiddlewarePoller(a.Poller, msgLogger)

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

func chance(c int) bool {
	return rand.Intn(99)+1 <= c
}

func (a *Aslince) handleCommand(text string, m *tb.Message) {
	// paint
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

	// plan
	if chetamRegex.MatchString(text) {
		status, err := a.plan.Status(m.Chat.ID)
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

	if strings.Contains(text, "рисуешь?") {
		a.Send(m.Chat, fmt.Sprintf("%d/100", a.paintChance), &tb.SendOptions{ReplyTo: m})
		return
	}
}

func (a *Aslince) answer(m *tb.Message) error {
	if a.talk != nil {
		text := a.talk.GenerateMessage(m.Text)
		_, err := a.Send(m.Chat, text, &tb.SendOptions{ReplyTo: m})
		if err != nil {
			return errors.Wrapf(err, "can't answer to message %d", m.ID)
		}
	}
	return nil
}

func (a *Aslince) handle(m *tb.Message) {
	a.lastMessage = m
	text := strings.ToLower(m.Text)
	if isCommand(text) {
		a.handleCommand(text, m)
		return
	} else if m.IsReply() && m.ReplyTo.Sender.ID == a.Me.ID || m.Private() {
		err := a.answer(m)
		if err != nil {
			log.Error("handle: ", err)
		}
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
		return
	}

	// цирк
	if strings.Contains(strings.ToLower(text), "цирк") {
		circus, err := os.Open("./resources/circus.jpg")
		if err != nil {
			log.Error(err)
			return
		}
		a.Send(m.Chat, &tb.Photo{File: tb.FromReader(circus)}, &tb.SendOptions{ReplyTo: m})
	}

	// eeeeeeeee boi
	if eeeeeBoiRegex.MatchString(strings.ToLower(text)) {
		err := a.eeeeeeBoi(m)
		if err != nil {
			log.Error("handle:", err)
			return
		}
	}

	err := a.planCheck(m)
	if err != nil {
		log.Error("handle: plan check error", err)
	}

	a.talk.Add(m.Text)
}

func (a *Aslince) eeeeeeBoi(m *tb.Message) error {
	eeeee, err := os.Open("./resources/eeeeeeee.ogg")
	if err != nil {
		return errors.Wrap(err, "eeeeeeBoi: can't open audio")
	}
	mcAndroid, err := os.Open("./resources/mc_android.jpg")
	if err != nil {
		return errors.Wrap(err, "eeeeeeBoi: can't open thumbnail")
	}
	_, err = a.Send(m.Chat, &tb.Audio{
		File:      tb.FromReader(eeeee),
		Duration:  1,
		Caption:   "",
		Thumbnail: &tb.Photo{File: tb.FromReader(mcAndroid)},
		Title:     "eeeeeeeeeeeeee boooooi",
		Performer: "mc android",
	}, &tb.SendOptions{ReplyTo: m})

	return errors.Wrap(err, "eeeeeeBoi: can't send message")
}

func (a *Aslince) planCheck(m *tb.Message) error {
	sources, err := a.plan.Check(m)
	if err != nil {
		return errors.Wrap(err, "planCheck: check error")
	}

	if bingo, err := a.plan.Bingo(m.Chat.ID); bingo {
		if err != nil {
			return errors.Wrap(err, "planCheck: error")
		}
		err := a.eeeeeeBoi(m)
		if err != nil {
			return errors.Wrap(err, "planCheck: bingo response error")
		}
	}

	if len(sources) > 0 {
		for _, s := range sources {
			_, err = a.Send(
				m.Chat,
				fmt.Sprintf("☑️ %s – чек", s), &tb.SendOptions{
					ReplyTo: m,
				})
			if err != nil {
				log.Error("handle: check response error", err)
			}
		}
	}
	return nil
}

func isCommand(text string) bool {
	return aslinceRegexp.MatchString(strings.ToLower(text))
}
