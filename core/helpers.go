package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	rc_models "github.com/RocketChat/Rocket.Chat.Go.SDK/models"

	"github.com/geekgonecrazy/rfd-tool/config"
)

func sendToRocketChat(text string) {
	if config.Config.RocketChatWebhook == "" {
		log.Println("Not sending webhook its not set")
		return
	}

	msg := &rc_models.PostMessage{
		Text: text,
	}

	go sendWebhoook(config.Config.RocketChatWebhook, msg)
}

func sendWebhoook(url string, payload *rc_models.PostMessage) {
	jsonText, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonText))
	if err != nil {
		fmt.Println(err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()
}
