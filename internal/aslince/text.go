package aslince

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/mb-14/gomarkov"
	log "github.com/sirupsen/logrus"
)

type msgList struct {
	Messages []msg `json:"messages"`
}

type msg struct {
	ID     int     `json:"id"`
	Type   string  `json:"type"`
	Date   string  `json:"date"`
	From   string  `json:"from"`
	FromID int     `json:"from_id"`
	Text   msgText `json:"text"`
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

func generateMessage(chain *gomarkov.Chain) string {
	tokens := []string{gomarkov.StartToken}
	for tokens[len(tokens)-1] != gomarkov.EndToken {
		next, _ := chain.Generate(tokens[(len(tokens) - 1):])
		tokens = append(tokens, next)
	}
	return strings.Join(tokens[1:len(tokens)-1], " ")
}

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

	for _, m := range l.Messages {
		if m.Text.Text != "" {
			log.Debug(m.ID, m.Text)
			chain.Add(strings.Split(m.Text.Text, " "))
		}
	}

	return chain, nil
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
