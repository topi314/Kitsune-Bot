package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/DisgoOrg/disgo"
	"github.com/DisgoOrg/disgo/api"
	"github.com/DisgoOrg/disgo/api/endpoints"
	"github.com/DisgoOrg/disgo/api/events"
)

var purrbotAPI = endpoints.NewCustomRoute(endpoints.GET, "https://purrbot.site/api/img/{nsfw/sfw}/{type}/{img/gif}")

type purrbotAPIRS struct {
	Error bool   `json:"error"`
	Link  string `json:"link"`
	Time  int    `json:"time"`
}

func main() {
	log.Infof("starting Kitsune-Bot...")

	dgo, err := disgo.NewBuilder(os.Getenv("kitsune-token")).
		SetLogLevel(log.InfoLevel).
		SetWebhookServerProperties("/webhooks/interactions/callback", 80, os.Getenv("kitsune-public-key")).
		AddEventListeners(&events.ListenerAdapter{OnSlashCommand: slashCommandListener}).
		Build()
	if err != nil {
		log.Fatalf("error while building disgo instance: %s", err)
		return
	}

	kitsuneCommand := api.SlashCommand{
		Name:        "kitsune",
		Description: "Sends a nice Kitsune",
	}

	if _, err = dgo.SetCommands(kitsuneCommand); err != nil {
		log.Errorf("error while registering commands: %s", err)
	}

	if err = dgo.Start(); err != nil {
		log.Fatalf("error while starting webhookserver: %s", err)
	}

	defer dgo.Close()

	log.Infof("Bot is now running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-s
}

func slashCommandListener(event *events.SlashCommandEvent) {
	if event.Name != "kitsune" {
		return
	}

	var rsBody purrbotAPIRS
	if err := event.Disgo.RestClient().Request(purrbotAPI.Compile("sfw", "kitsune", "img"), nil, &rsBody); err != nil {
		log.Errorf("error retrieving kitsune: %s", err)
		if err = event.Reply(api.NewInteractionResponseBuilder().
			SetContent("Sowy I have trouble reaching my Kitsune API ≧ ﹏ ≦").
			SetEphemeral(true).
			Build(),
		); err != nil {
			log.Errorf("error sending reply: %s", err)
		}
	}
	if err := event.Reply(api.NewInteractionResponseBuilder().
		SetEmbeds(api.NewEmbedBuilder().
			SetColor(16777215).
			SetImage(&rsBody.Link).
			Build(),
		).Build(),
	); err != nil {
		log.Errorf("error sending reply: %s", err)
	}
}
