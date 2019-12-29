package main

import (
	"context"
	"fmt"
	"log"
	"os"

	dg "cloud.google.com/go/dialogflow/apiv2"
	"golang.org/x/net/websocket"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
)

type Message struct {
	Author string `json:"author"`
	Body   string `json:"body"`
}

func main() {
	// configuration
	credentialsFile := os.Getenv("CREDENTIALS_FILE")
	if credentialsFile == "" {
		log.Fatal("set CREDENTIALS_FILE")
	}
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		log.Fatal("set PROJECT_ID")
	}
	originalURL := os.Getenv("ORIGINAL_URL")
	if originalURL == "" {
		log.Fatal("set ORIGINAL_URL")
	}
	wsURL := os.Getenv("WS_URL")
	if wsURL == "" {
		log.Fatal("set WS_URL")
	}

	// dialogflow client
	// https://dialogflow.com/docs/reference/v2-auth-setup
	client, err := dg.NewSessionsClient(context.Background(), option.WithCredentialsFile(`key.json`))
	if err != nil {
		log.Fatal(err)
	}

	// websocket connect
	conn, err := websocket.Dial(wsURL, "", originalURL)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	var msg, sendMessage Message
	for {
		// get message
		if err := websocket.JSON.Receive(conn, &msg); err != nil {
			log.Println(err)
			break
		}
		// not answer on bot message
		if msg.Body == sendMessage.Body {
			continue
		}

		// get dialogflow answer
		resp, err := client.DetectIntent(context.Background(), &dialogflow.DetectIntentRequest{
			Session: fmt.Sprintf("projects/%s/agent/sessions/%s'", projectID, msg.Author),
			QueryInput: &dialogflow.QueryInput{
				Input: &dialogflow.QueryInput_Text{Text: &dialogflow.TextInput{
					Text:         msg.Body,
					LanguageCode: "ru",
				}},
			},
		})
		if err != nil {
			log.Println(err)
			break
		}
		result := resp.GetQueryResult().FulfillmentText
		if result == "" {
			sendMessage = Message{Body: ` (бот) Чего? Не понимаю тебя!`}
		} else {
			sendMessage = Message{Body: ` (бот) ` + result}
		}

		if err = websocket.JSON.Send(conn, &sendMessage); err != nil {
			log.Println(err)
			break
		}
	}
}
