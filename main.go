package main

import (
	"context"
	"fmt"
	"net/url"
	"time"

	dg "cloud.google.com/go/dialogflow/apiv2"
	"github.com/gorilla/websocket"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
)

type config struct {
	CredentialsJSON string `envconfig:"CREDENTIALS_JSON"`
	ProjectID       string `envconfig:"PROJECT_ID"`
	WSUrl           string `envconfig:"WS_URL" default:"wss://chat.delgus.com/entry"`
}

// Message for comunication
type Message struct {
	Author string `json:"author"`
	Body   string `json:"body"`
}

func newConn(u string) *websocket.Conn {
	for {
		conn, _, err := websocket.DefaultDialer.Dial(u, nil)
		if err != nil {
			// hardcore reconnect
			time.Sleep(10 * time.Second)
			continue
		}
		return conn
	}
}

func main() {
	// configuration
	var cfg config
	if err := envconfig.Process("", &cfg); err != nil {
		logrus.Fatal("can not get configuration")
	}

	// dialogflow client
	// https://dialogflow.com/docs/reference/v2-auth-setup
	client, err := dg.NewSessionsClient(
		context.Background(),
		option.WithCredentialsJSON([]byte(cfg.CredentialsJSON)))
	if err != nil {
		logrus.Fatal(err)
	}

	u, err := url.Parse(cfg.WSUrl)
	if err != nil {
		logrus.Fatal("not correct url")
	}

	// websocket connect
	conn := newConn(u.String())

	var msg, sendMessage Message
	for {
		// get message
		if err := conn.ReadJSON(&msg); err != nil {
			conn.Close()
			logrus.Error(err)
			conn = newConn(u.String())
		}

		// not answer on bot message
		if msg.Body == sendMessage.Body {
			continue
		}

		// get dialogflow answer
		resp, err := client.DetectIntent(context.Background(), &dialogflow.DetectIntentRequest{
			Session: fmt.Sprintf("projects/%s/agent/sessions/%s'", cfg.ProjectID, msg.Author),
			QueryInput: &dialogflow.QueryInput{
				Input: &dialogflow.QueryInput_Text{Text: &dialogflow.TextInput{
					Text:         msg.Body,
					LanguageCode: "ru",
				}},
			},
		})
		if err != nil {
			logrus.Error(err)
			break
		}
		result := resp.GetQueryResult().FulfillmentText
		if result == "" {
			sendMessage = Message{Body: `Чего? Не понимаю тебя!`}
		} else {
			sendMessage = Message{Body: result}
		}

		if err = conn.WriteJSON(&sendMessage); err != nil {
			logrus.Error(err)
			conn.Close()
			conn = newConn(u.String())
		}
	}
}
