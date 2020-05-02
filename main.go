package main

import (
	"context"
	"fmt"

	dg "cloud.google.com/go/dialogflow/apiv2"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
)

type config struct {
	CredentialsJSON string `envconfig:"CREDENTIALS_JSON"`
	ProjectID       string `envconfig:"PROJECT_ID"`
	WSUrl           string `envconfig:"WS_URL"`
	OriginalURL     string `envconfig:"ORIGINAL_URL"`
}

// Message for comunication
type Message struct {
	Author string `json:"author"`
	Body   string `json:"body"`
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

	// websocket connect
	conn, err := websocket.Dial(cfg.WSUrl, "", cfg.OriginalURL)
	if err != nil {
		logrus.Fatal(err)
	}
	defer conn.Close()

	var msg, sendMessage Message
	for {
		// get message
		if err := websocket.JSON.Receive(conn, &msg); err != nil {
			logrus.Error(err)
			break
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
			sendMessage = Message{Body: ` (бот) Чего? Не понимаю тебя!`}
		} else {
			sendMessage = Message{Body: ` (бот) ` + result}
		}

		if err = websocket.JSON.Send(conn, &sendMessage); err != nil {
			logrus.Error(err)
			break
		}
	}
}
