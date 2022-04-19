package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/httpserver"
	"github.com/disgoorg/disgo/rest/route"
	"github.com/disgoorg/dislog"
	"github.com/disgoorg/snowflake"
	"github.com/sirupsen/logrus"
)

const (
	embedColor = 0xFC9803
	version    = "dev"
)

var (
	//go:embed senko.png
	senkoImage []byte

	logWebhookID    = snowflake.GetSnowflakeEnv("log_webhook_id")
	logWebhookToken = os.Getenv("log_webhook_token")
	token           = os.Getenv("kitsune_token")
	publicKey       = os.Getenv("kitsune_public_key")
	logLevel, _     = logrus.ParseLevel(os.Getenv("log_level"))

	purrbotAPI   = route.NewCustomAPIRoute(route.GET, "https://purrbot.site/api", "/img/{nsfw/sfw}/{type}/{img/gif}")
	randomfoxAPI = route.NewCustomAPIRoute(route.GET, "https://randomfox.ca", "/{type}")

	commands = []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			CommandName:       "kitsune",
			Description:       "Sends a nice random Kitsune",
			DefaultPermission: true,
		},
		discord.SlashCommandCreate{
			CommandName:       "senko",
			Description:       "Sends a nice random Senko",
			DefaultPermission: true,
		},
		discord.SlashCommandCreate{
			CommandName:       "fox",
			Description:       "Sends a nice random Fox",
			DefaultPermission: true,
		},
		discord.SlashCommandCreate{
			CommandName:       "info",
			Description:       "Sends some info about me",
			DefaultPermission: true,
		},
	}
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logLevel)
	if logWebhookID != "" && logWebhookToken != "" {
		dlog, err := dislog.New(dislog.WithLogger(logger), dislog.WithWebhookIDToken(logWebhookID, logWebhookToken), dislog.WithLogLevels(dislog.InfoLevelAndAbove...))
		if err != nil {
			logger.Fatal("error initializing dislog %s", err)
		}
		defer dlog.Close(context.TODO())
		logger.AddHook(dlog)
	}
	logger.Infof("starting Kitsune-Bot...")

	client, err := disgo.New(token,
		bot.WithLogger(logger),
		bot.WithCacheConfigOpts(
			cache.WithCacheFlags(cache.FlagsNone),
			cache.WithMemberCachePolicy(cache.MemberCachePolicyNone),
			cache.WithMessageCachePolicy(cache.MessageCachePolicyNone),
		),
		bot.WithHTTPServerConfigOpts(
			httpserver.WithAddress(":80"),
			httpserver.WithPublicKey(publicKey),
			httpserver.WithURL("/webhooks/interactions/callback"),
		),
		bot.WithEventListeners(&events.ListenerAdapter{
			OnApplicationCommandInteraction: commandListener,
		}),
	)
	if err != nil {
		logger.Fatalf("error while building disgo instance: %s", err)
		return
	}

	if _, err = client.Rest().Applications().SetGlobalCommands(client.ApplicationID(), commands); err != nil {
		logger.Error("error while registering commands: ", err)
	}

	if err = client.StartHTTPServer(); err != nil {
		logger.Error("error while starting http server: ", err)
	}

	defer client.Close(context.TODO())

	logger.Info("Bot is now running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-s
}

func commandListener(e *events.ApplicationCommandInteractionEvent) {
	var (
		imageLink    string
		errorMessage string
	)
	switch name := e.Data.CommandName(); name {
	case "kitsune", "senko":
		if err := e.DeferCreateMessage(false); err != nil {
			e.Client().Logger().Error("error while deferring message creation: ", err)
			return
		}
		compiledRoute, _ := purrbotAPI.Compile(nil, "sfw", name, "img")
		var rsBody purrbotAPIResponse
		if err := e.Client().Rest().RestClient().Do(compiledRoute, nil, &rsBody); err != nil {
			e.Client().Logger().Error("error retrieving kitsune or senko: ", err)
			errorMessage = "Sowy I had trouble reaching my " + name + " API ≧ ﹏ ≦"
		} else {
			imageLink = rsBody.Link
		}

	case "fox":
		if err := e.DeferCreateMessage(false); err != nil {
			e.Client().Logger().Error("error while deferring message creation: ", err)
			return
		}
		compiledRoute, _ := randomfoxAPI.Compile(nil, "floof")
		var rsBody randomfoxAPIResponse
		if err := e.Client().Rest().RestClient().Do(compiledRoute, nil, &rsBody); err != nil {
			e.Client().Logger().Error("error retrieving fox: ", err)
			errorMessage = "Sowy I had trouble reaching my Fox API ≧ ﹏ ≦"
		} else {
			imageLink = rsBody.Image
		}

	case "info":
		if err := e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{
				discord.NewEmbedBuilder().
					SetDescription("Hi, I'm a small bot which delivers you Kitsune, Senko and Fox images./nI hope you enjoy the images.").
					AddField("Version", version, false).
					SetColor(embedColor).
					SetThumbnail("attachment://senko.png").
					Build(),
			},
			Files: []*discord.File{
				discord.NewFile("senko.png", "Senko", bytes.NewBuffer(senkoImage)),
			},
			Components: []discord.ContainerComponent{discord.NewActionRow(
				discord.NewLinkButton("GitHub", "https://github.com/TopiSenpai/Kitsune-Bot"),
				discord.NewLinkButton("Discord", "https://discord.gg/sD3ABd5"),
				discord.NewLinkButton("Invite Me", fmt.Sprintf("https://discord.com/oauth2/authorize?client_id=%s&scope=applications.commands", e.Client().ID())),
			)},
		}); err != nil {
			e.Client().Logger().Error("error while sending info message: ", err)
		}
		return

	default:
		e.Client().Logger().Warn("unknown command with name %s received", name)
		return
	}

	var messageUpdate discord.MessageUpdate

	if errorMessage != "" {
		messageUpdate = discord.MessageUpdate{Content: &errorMessage}
	} else {
		messageUpdate = discord.MessageUpdate{Embeds: &[]discord.Embed{
			{
				Color: embedColor,
				Image: &discord.EmbedResource{
					URL: imageLink,
				},
			},
		}}
	}

	if _, err := e.Client().Rest().Interactions().UpdateInteractionResponse(e.ApplicationID(), e.Token(), messageUpdate); err != nil {
		e.Client().Logger().Error("error updating interaction: ", err)
	}
}

type purrbotAPIResponse struct {
	Error bool   `json:"error"`
	Link  string `json:"link"`
	Time  int    `json:"time"`
}

type randomfoxAPIResponse struct {
	Image string `json:"image"`
	Link  string `json:"link"`
}
