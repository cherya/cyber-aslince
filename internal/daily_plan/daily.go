package daily_plan

import (
	"fmt"
	"strings"
	"time"

	"github.com/cherya/cyber-aslince/internal/messages_helpers"
	"github.com/cherya/cyber-aslince/internal/time_helpers"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
)

const tassID = 418166693
const sberID = 95123848

var sources = map[string][]string{
	"зе флоу":       {"the-flow", "The Flow"},
	"тасс":          {"tass", "tass.ru", "ТАСС"},
	"двач":          {"2ch", "двач"},
	"мдк":           {"mdk", "мдк"},
	"рифмы и панчи": {"рифмы и панчи"},
	"сбер":          {},
}

type DailyPlan struct {
	redis     *redis.Pool
	namespace string
}

func New(r *redis.Pool, namespase string) *DailyPlan {
	return &DailyPlan{
		redis:     r,
		namespace: namespase,
	}
}

func (d *DailyPlan) Status(chatID int64) (string, error) {
	conn := d.redis.Get()
	defer conn.Close()
	b := strings.Builder{}
	for s := range sources {
		count, err := redis.Int(conn.Do("GET", dailySrcKey(d.namespace, chatID, s)))
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

func (d *DailyPlan) Bingo(chatID int64) (bool, error) {
	conn := d.redis.Get()
	defer conn.Close()

	for s := range sources {
		val, err := redis.Int(conn.Do("GET", dailySrcKey(d.namespace, chatID, s)))
		if err != nil || val < 1 {
			return false, nil
		}
	}

	set, err := redis.String(conn.Do("SET", dailySrcKey(d.namespace, chatID, "bingo"), 0, "NX", "EX", (time.Hour * 24).Seconds()))
	if err != nil {
		return false, errors.Wrap(err, "Bingo: can't set bingo")
	}
	if set != "OK" {
		log.Info("bingo++")
		return false, nil
	}

	return true, nil
}

func (d *DailyPlan) Check(m *tb.Message) ([]string, error) {
	successSources := make(map[string]struct{}, 0)
	for source, checks := range sources {
		if checkLinks(m, checks) {
			successSources[source] = struct{}{}
		}
		if m.IsForwarded() && m.OriginalChat != nil && checkText(m.OriginalChat.Title, checks){
			successSources[source] = struct{}{}
		}
	}
	if m.Sender.ID == tassID {
		successSources["тасс"] = struct{}{}
	}
	if m.Sender.ID == sberID {
		successSources["сбер"] = struct{}{}
	}

	conn := d.redis.Get()
	defer conn.Close()

	for s := range successSources {
		key := dailySrcKey(d.namespace, m.Chat.ID, s)
		_, err := conn.Do("SET", key, 0, "NX", "EX", (time.Hour * 24).Seconds())
		if err != nil {
			return nil, errors.Wrap(err, "Check: can't set source")
		}
		count, err := redis.Int(conn.Do("INCR", key))
		if err != nil {
			return nil, errors.Wrap(err, "Check: can't incr source")
		}
		if count > 1 {
			delete(successSources, s)
		}
	}

	ss := make([]string, 0, len(successSources))
	for s := range successSources {
		ss = append(ss, s)
	}

	return ss, nil
}

func dailySrcKey(namespace string, chatID int64, src string) string {
	t, err := time_helpers.TimeIn(time.Now(), "Europe/Moscow")
	if err != nil {
		t = time.Now()
	}
	return fmt.Sprintf("%s:%d:%s:%s", namespace, chatID, src, t.Format("02-01-2006"))
}

func checkLinks(m *tb.Message, checks []string) bool {
	for _, e := range m.Entities {
		var text string
		if e.Type == tb.EntityURL {
			text = messages_helpers.TextFromMsg(m)[e.Offset : e.Offset+e.Length]
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
