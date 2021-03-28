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
var randomfoxAPI = endpoints.NewCustomRoute(endpoints.GET, "https://randomfox.ca/{type}")

type purrbotAPIRS struct {
	Error bool   `json:"error"`
	Link  string `json:"link"`
	Time  int    `json:"time"`
}

type randomfoxAPIRS struct {
	Image bool   `json:"image"`
	Link  string `json:"link"`
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
		Description: "Sends a nice random Kitsune",
	}
	senkoCommand := api.SlashCommand{
		Name:        "senko",
		Description: "Sends a nice random Senko",
	}
	foxCommand := api.SlashCommand{
		Name:        "fox",
		Description: "Sends a nice random Fox",
	}

	if _, err = dgo.SetCommands(kitsuneCommand, senkoCommand, foxCommand); err != nil {
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
	var link string
	var errStr string
	switch event.Name {
	case "kitsune", "senko":
		var rsBody purrbotAPIRS
		if err := event.Disgo.RestClient().Request(purrbotAPI.Compile("sfw", event.Name, "img"), nil, &rsBody); err != nil {
			log.Errorf("error retrieving kitsune or senko: %s", err)
			errStr = "Sowy I have trouble reaching my " + event.Name + " API ≧ ﹏ ≦"
		} else {
			link = rsBody.Link
		}
	case "fox":
		var rsBody randomfoxAPIRS
		if err := event.Disgo.RestClient().Request(randomfoxAPI.Compile("floof"), nil, &rsBody); err != nil {
			log.Errorf("error retrieving fox: %s", err)
			errStr = "Sowy I have trouble reaching my Fox API ≧ ﹏ ≦"
		} else {
			link = rsBody.Link
		}
	default:
		return
	}

	if errStr != "" {
		if err := event.Reply(api.NewInteractionResponseBuilder().
			SetContent(errStr).
			SetEphemeral(true).
			Build(),
		); err != nil {
			log.Errorf("error sending reply: %s", err)
		}
		return
	}

	if err := event.Reply(api.NewInteractionResponseBuilder().
		SetEmbeds(api.NewEmbedBuilder().
			SetColor(16564739).
			SetImage(&link).
			Build(),
		).Build(),
	); err != nil {
		log.Errorf("error sending reply: %s", err)
	}
}
