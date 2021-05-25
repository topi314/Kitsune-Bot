package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/DisgoOrg/dislog"
	"github.com/sirupsen/logrus"

	"github.com/DisgoOrg/disgo"
	"github.com/DisgoOrg/disgo/api"
	"github.com/DisgoOrg/disgo/api/endpoints"
	"github.com/DisgoOrg/disgo/api/events"
)

var logWebhookToken = os.Getenv("log_webhook_token")
var token = os.Getenv("kitsune-token")
var publicKey = os.Getenv("kitsune-public-key")

var purrbotAPI = endpoints.NewCustomRoute(endpoints.GET, "https://purrbot.site/api/img/{nsfw/sfw}/{type}/{img/gif}")
var randomfoxAPI = endpoints.NewCustomRoute(endpoints.GET, "https://randomfox.ca/{type}")

type purrbotAPIRS struct {
	Error bool   `json:"error"`
	Link  string `json:"link"`
	Time  int    `json:"time"`
}

type randomfoxAPIRS struct {
	Image string `json:"image"`
	Link  string `json:"link"`
}

var logger = logrus.New()

func main() {
	httpClient := http.DefaultClient
	logger.SetLevel(logrus.InfoLevel)
	dlog, err := dislog.NewDisLogByToken(httpClient, logrus.InfoLevel, logWebhookToken, dislog.InfoLevelAndAbove...)
	if err != nil {
		logger.Errorf("error initializing dislog %s", err)
		return
	}
	defer dlog.Close()

	logger.AddHook(dlog)
	logger.Infof("starting Kitsune-Bot...")

	dgo, err := disgo.NewBuilder(token).
		SetLogger(logger).
		SetHTTPClient(httpClient).
		SetCacheFlags(api.CacheFlagsNone).
		SetMemberCachePolicy(api.MemberCachePolicyNone).
		SetMessageCachePolicy(api.MessageCachePolicyNone).
		SetWebhookServerProperties("/webhooks/interactions/callback", 80, publicKey).
		AddEventListeners(&events.ListenerAdapter{OnSlashCommand: slashCommandListener}).
		Build()
	if err != nil {
		log.Fatalf("error while building disgo instance: %s", err)
		return
	}

	commands := []*api.CommandCreate{
		{
			Name:        "kitsune",
			Description: "Sends a nice random Kitsune",
		},
		{
			Name:        "senko",
			Description: "Sends a nice random Senko",
		},
		{
			Name:        "fox",
			Description: "Sends a nice random Fox",
		},
	}

	if _, err = dgo.SetCommands(commands...); err != nil {
		logger.Errorf("error while registering commands: %s", err)
	}

	dgo.Start()

	defer dgo.Close()

	logger.Infof("Bot is now running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-s
}

func slashCommandListener(event *events.SlashCommandEvent) {
	var link string
	var errStr string
	switch event.CommandName {
	case "kitsune", "senko":
		compiledRoute, _ := purrbotAPI.Compile("sfw", event.CommandName, "img")
		var rsBody purrbotAPIRS
		if err := event.Disgo().RestClient().Request(compiledRoute, nil, &rsBody); err != nil {
			logger.Errorf("error retrieving kitsune or senko: %s", err)
			errStr = "Sowy I have trouble reaching my " + event.CommandName + " API ≧ ﹏ ≦"
		} else {
			link = rsBody.Link
		}
	case "fox":
		compiledRoute, _ := randomfoxAPI.Compile("floof")
		var rsBody randomfoxAPIRS
		if err := event.Disgo().RestClient().Request(compiledRoute, nil, &rsBody); err != nil {
			logger.Errorf("error retrieving fox: %s", err)
			errStr = "Sowy I have trouble reaching my Fox API ≧ ﹏ ≦"
		} else {
			link = rsBody.Image
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
			logger.Errorf("error sending reply: %s", err)
		}
		return
	}

	if err := event.Reply(api.NewInteractionResponseBuilder().
		SetEmbeds(api.NewEmbedBuilder().
			SetColor(16564739).
			SetImage(link).
			Build(),
		).Build(),
	); err != nil {
		logger.Errorf("error sending reply: %s", err)
	}
}
