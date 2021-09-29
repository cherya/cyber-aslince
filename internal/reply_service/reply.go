package replyservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type GeneratorResponse struct {
	Text    string `json:"text"`
	Sticker string `json:"sticker"`
}

type Request struct {
	Text   string
	ID     int
	ChatID int64
}

type Reply struct {
	Response GeneratorResponse
	ID       int
	ChatID   int64
}

type Generator struct {
	requests []Request
	replies  chan Reply
	baseURL  string
	client   *http.Client
}

func New(url string) *Generator {
	g := &Generator{
		requests: make([]Request, 0, 5),
		replies:  make(chan Reply, 10),
		baseURL:  url,
		client: &http.Client{
			Transport: &http.Transport{
				MaxConnsPerHost: 1,
			},
			Timeout: 60 * time.Second,
		},
	}

	go func() {
		for {
			if len(g.requests) == 0 {
				time.Sleep(time.Second)
			} else {
				var req Request
				req, g.requests = g.requests[0], g.requests[1:]

				resp, err := g.doRequest(req.Text)
				if err != nil {
					log.Error(err)
					continue
				}

				g.replies <- Reply{
					Response: *resp,
					ID:       req.ID,
					ChatID:   req.ChatID,
				}
			}
		}
	}()

	return g
}

func (g *Generator) doRequest(text string) (*GeneratorResponse, error) {
	reqBody := strings.NewReader(fmt.Sprintf(`{"text":"%s"}`, text))

	resp, err := g.client.Post(g.baseURL, "application/json", reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "request: error doing generator request")
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var data GeneratorResponse
	err = decoder.Decode(&data)
	if err != nil {
		return nil, errors.Wrap(err, "request: error decoding generator response")
	}
	return &data, nil
}

func (g *Generator) Generate(text string, replyTo int, chatID int64) {
	if len(g.requests) == 5 {
		g.requests = g.requests[1:]
	}
	g.requests = append(g.requests, Request{
		Text:   text,
		ID:     replyTo,
		ChatID: chatID,
	})
}

func (g *Generator) RepliesChan() chan Reply {
	return g.replies
}
