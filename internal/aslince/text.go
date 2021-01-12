package aslince

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"regexp"
	"strings"

	"github.com/mb-14/gomarkov"
	log "github.com/sirupsen/logrus"
)

type msgList struct {
	Messages []msg `json:"messages"`
}

type msg struct {
	ID               int     `json:"id"`
	Type             string  `json:"type"`
	Date             string  `json:"date"`
	From             string  `json:"from"`
	FromID           int     `json:"from_id"`
	Text             msgText `json:"text"`
	ForwardedFrom    string  `json:"forwarded_from"`
	ViaBot           string  `json:"via_bot"`
	ReplyToMessageID int     `json:"reply_to_message_id"`
}

type msgText struct {
	Text string
}

func (m *msgText) UnmarshalJSON(data []byte) error {
	var ar = make([]interface{}, 0)
	err := json.Unmarshal(data, &ar)
	if err != nil {
		m.Text = string(data)
		return nil
	}
	m.Text = ""
	return nil
}

type Talker struct {
	messages map[int]msg
	replies  map[int][]int
	chain    *gomarkov.Chain
}

func NewTalker(chatExportData []byte, model []byte) (*Talker, error) {
	t := &Talker{
		messages: make(map[int]msg),
		replies:  make(map[int][]int),
	}
	var err error
	var emptyModel = false
	if len(model) != 0 {
		log.Info("loading model")
		t.chain, err = loadModel(model)
		if err != nil {
			log.Error("error loading model", err)
			t.chain = gomarkov.NewChain(1)
			emptyModel = true
		} else {
			log.Info("model successfully loaded")
		}
	} else {
		log.Info("model is empty, creating")
		t.chain = gomarkov.NewChain(1)
		emptyModel = true
	}

	var l msgList
	err = json.Unmarshal(chatExportData, &l)
	if err != nil {
		return nil, err
	}
	var amount = 0
	for _, m := range l.Messages {
		t.messages[m.ID] = m
		if m.ReplyToMessageID != 0 {
			_, ok := t.replies[m.ReplyToMessageID]
			if !ok {
				t.replies[m.ReplyToMessageID] = make([]int, 0)
			}
			t.replies[m.ReplyToMessageID] = append(t.replies[m.ReplyToMessageID], m.ID)
		}
		if m.ForwardedFrom != "" || m.ViaBot != "" || m.Type != "message" {
			continue
		}
		text := cleanText(m.Text.Text)
		if emptyModel && text != "" && text != " " {
			amount++
			t.chain.Add(strings.Split(text, " "))
		}
	}
	log.Infof("Added %d messages to chain", amount)
	return t, nil
}

func (t *Talker) GenerateMessage(text string) string {
	tok := cleanText(t.getFirstToken(text))
	tokens := []string{gomarkov.StartToken, tok}
	for tokens[len(tokens)-1] != gomarkov.EndToken {
		next, err := t.chain.Generate(tokens[(len(tokens) - 1):])
		if err != nil {
			log.Errorf("error generating text. token='%s'. %s", tok, err)
			tokens = []string{gomarkov.StartToken}
			continue
		}
		if next == gomarkov.EndToken && len(tokens) < 3 {
			tokens = []string{gomarkov.StartToken}
			continue
		}
		tokens = append(tokens, next)
	}
	result := strings.Join(tokens[1:len(tokens)-1], " ")
	result = strings.ReplaceAll(result, gomarkov.StartToken, "")
	result = strings.ReplaceAll(result, gomarkov.EndToken, "")
	return result
}

func (t *Talker) GetModel() *gomarkov.Chain {
	return t.chain
}

func (t *Talker) Add(text string) {
	if text != "" && text != " " {
		t.chain.Add(strings.Split(cleanText(text), " "))
	}
}

func (t *Talker) getFirstToken(text string) string {
	s := strings.Split(text, " ")
	w := s[rand.Intn(len(s))]
	if len(t.messages) == 0 {
		log.Debug("getFirstToken: empty messages")
		return w
	}
	for _, m := range t.messages {
		if strings.Contains(m.Text.Text, w) {
			if r, ok := t.replies[m.ID]; ok {
				rid := r[rand.Intn(len(r))]
				return strings.Split(t.messages[rid].Text.Text, " ")[0]
			}
		}
	}
	return w
}

var (
	r1       = regexp.MustCompile(`[^a-zA-Zа-яА-ЯёЁ+.0-9\s%]`)
	r2       = regexp.MustCompile(`(,:\.)`)
	space    = regexp.MustCompile(`\s+`)
	username = regexp.MustCompile(`@[a-zA-Z0-9]+`)
)

func cleanText(text string) string {
	text = strings.ReplaceAll(text, `\n`, " ")
	text = username.ReplaceAllString(text, "")
	text = r1.ReplaceAllString(text, " ")
	text = r2.ReplaceAllStringFunc(text, func(m string) string {
		return fmt.Sprintf(" %s ", m)
	})

	text = space.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	return strings.ToLower(text)
}

func saveModel(c *gomarkov.Chain) error {
	jsonObj, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("model.json", jsonObj, 0644)
}

func loadModel(data []byte) (*gomarkov.Chain, error) {
	var chain gomarkov.Chain
	err := json.Unmarshal(data, &chain)
	if err != nil {
		return nil, err
	}
	return &chain, nil
}
