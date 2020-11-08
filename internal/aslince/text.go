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

func getFirstToken(text string) string {
	if len(messages) == 0 {
		return ""
	}
	s := strings.Split(text, " ")
	w := s[rand.Intn(len(s))]
	for _, m := range messages {
		if strings.Contains(m.Text.Text, w) {
			if r, ok := replies[m.ID]; ok {
				rid := r[rand.Intn(len(r))]
				return strings.Split(messages[rid].Text.Text, " ")[0]
			}
		}
	}
	return w
}

func generateMessage(chain *gomarkov.Chain, text string) string {
	t := getFirstToken(text)
	tokens := []string{gomarkov.StartToken, cleanText(t)}
	for tokens[len(tokens)-1] != gomarkov.EndToken {
		next, err := chain.Generate(tokens[(len(tokens) - 1):])
		if err != nil {
			log.Error("error generating text ", err)
			return strings.Join(tokens, " ")
		}
		tokens = append(tokens, next)
	}
	result := strings.Join(tokens[1:len(tokens)-1], " ")
	result = strings.ReplaceAll(result, gomarkov.StartToken, "")
	result = strings.ReplaceAll(result, gomarkov.EndToken, "")
	return result
}

var (
	messages map[int]msg   = make(map[int]msg)
	replies  map[int][]int = make(map[int][]int)
)

func buildModel() (*gomarkov.Chain, error) {
	chain := gomarkov.NewChain(1)
	data, err := ioutil.ReadFile("./opg/result.json")
	if err != nil {
		return nil, err
	}
	var l msgList
	err = json.Unmarshal(data, &l)
	if err != nil {
		return nil, err
	}
	var amount = 0
	for _, m := range l.Messages {
		if m.ForwardedFrom != "" || m.ViaBot != "" || m.Type != "message" {
			continue
		}
		text := cleanText(m.Text.Text)
		if text != "" {
			chain.Add(strings.Split(text, " "))
		}
		messages[m.ID] = m
		if m.ReplyToMessageID != 0 {
			_, ok := replies[m.ReplyToMessageID]
			if !ok {
				replies[m.ReplyToMessageID] = make([]int, 0)
			}
			replies[m.ReplyToMessageID] = append(replies[m.ReplyToMessageID], m.ID)
		}
		amount++
	}
	log.Infof("Added %d messages to chain", amount)
	return chain, nil
}

var (
	r1    = regexp.MustCompile(`[\"\'\(\)\[\]\\\/\?\!\-_=—,]`)
	r2    = regexp.MustCompile(`(\,\:)`)
	space = regexp.MustCompile(`\s+`)
)

func cleanText(text string) string {
	text = strings.ReplaceAll(text, `\n`, "")
	text = r1.ReplaceAllString(text, "")
	text = r2.ReplaceAllStringFunc(text, func(m string) string {
		return fmt.Sprintf(" %s ", m)
	})
	text = space.ReplaceAllString(text, " ")
	return strings.ToLower(text)
}

func saveModel(c *gomarkov.Chain) error {
	jsonObj, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("model.json", jsonObj, 0644)
}

func loadModel() (*gomarkov.Chain, error) {
	var chain gomarkov.Chain
	data, err := ioutil.ReadFile("model.json")
	if err != nil {
		return &chain, err
	}
	err = json.Unmarshal(data, &chain)
	if err != nil {
		return &chain, err
	}
	return &chain, nil
}
