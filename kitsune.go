package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/httpserver"
	"github.com/disgoorg/dislog"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sirupsen/logrus"
)

const embedColor = 0xFC9803

var (
	//go:embed senko.png
	senkoImage []byte

	logWebhookID    = snowflake.GetEnv("log_webhook_id")
	logWebhookToken = os.Getenv("log_webhook_token")
	token           = os.Getenv("kitsune_token")
	publicKey       = os.Getenv("kitsune_public_key")
	logLevel, _     = logrus.ParseLevel(os.Getenv("log_level"))

	animatedTypes = map[string]bool{
		"kitsune": false,
		"senko":   false,
		"shiro":   false,
		"tail":    true,
		"fluff":   true,
	}

	logger   = logrus.New()
	commands = []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        "kitsune",
			Description: "Sends a nice random Kitsune",
		},
		discord.SlashCommandCreate{
			Name:        "senko",
			Description: "Sends a nice random Senko",
		},
		discord.SlashCommandCreate{
			Name:        "shiro",
			Description: "Sends a nice random Shiro",
		},
		discord.SlashCommandCreate{
			Name:        "tail",
			Description: "Sends a nice random fox tail",
		},
		discord.SlashCommandCreate{
			Name:        "fluff",
			Description: "Sends a nice random fox fluff",
		},
		discord.SlashCommandCreate{
			Name:        "fox",
			Description: "Sends a nice random Fox",
		},
		discord.SlashCommandCreate{
			Name:        "info",
			Description: "Sends some info about me",
		},
	}
)

func main() {
	logger.SetLevel(logLevel)
	if logWebhookID != 0 && logWebhookToken != "" {
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
		bot.WithHTTPServerConfigOpts(publicKey,
			httpserver.WithAddress(":80"),
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

	if _, err = client.Rest().SetGlobalCommands(client.ApplicationID(), commands); err != nil {
		logger.Error("error while registering commands: ", err)
	}

	if err = client.OpenHTTPServer(); err != nil {
		logger.Error("error while starting http server: ", err)
	}

	defer client.Close(context.TODO())

	logger.Info("Bot is now running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-s
}

func commandListener(e *events.ApplicationCommandInteractionCreate) {
	var imageLink string
	switch name := e.Data.CommandName(); name {
	case "kitsune", "senko", "shiro", "fluff", "tail":
		if err := e.DeferCreateMessage(false); err != nil {
			logger.Error("error while deferring message creation: ", err)
			return
		}

		rs, err := http.Get(purrbotAPIURL(name, animatedTypes[name]))
		if err != nil || rs.StatusCode != http.StatusOK {
			logger.Error("error retrieving kitsune or senko: ", err)
			updateInteraction(e, discord.MessageUpdate{
				Content: json.Ptr("Sowy I had trouble reaching my " + name + " API ≧ ﹏ ≦"),
			})
			return
		}
		defer rs.Body.Close()

		var v purrbotAPIResponse
		if err = json.NewDecoder(rs.Body).Decode(&v); err != nil {
			logger.Error("error decoding kitsune or senko response: ", err)
			updateError(e, "Sowy I had trouble decoding my "+name+" API ≧ ﹏ ≦")
			return
		}
		imageLink = v.Link

	case "fox":
		if err := e.DeferCreateMessage(false); err != nil {
			logger.Error("error while deferring message creation: ", err)
			return
		}

		rs, err := http.Get(randomFoxAPIURL)
		if err != nil || rs.StatusCode != http.StatusOK {
			logger.Error("error retrieving fox: ", err)
			updateError(e, "Sowy I had trouble reaching my Fox API ≧ ﹏ ≦")
			return
		}

		var v randomfoxAPIResponse
		if err = json.NewDecoder(rs.Body).Decode(&v); err != nil {
			logger.Error("error decoding kitsune or senko response: ", err)
			updateError(e, "Sowy I had trouble decoding my "+name+" API ≧ ﹏ ≦")
			return
		}
		imageLink = v.Image

	case "info":
		if err := e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{
				discord.NewEmbedBuilder().
					SetDescription("Hi, I'm a small bot which delivers you Kitsune, Senko and Fox images./nI hope you enjoy the images.").
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
			logger.Error("error while sending info message: ", err)
		}
		return

	default:
		logger.Warn("unknown command with name %s received", name)
		updateError(e, "Sowy I don't know this command ≧ ﹏ ≦")
		return
	}

	updateInteraction(e, discord.MessageUpdate{Embeds: &[]discord.Embed{
		{
			Color: embedColor,
			Image: &discord.EmbedResource{
				URL: imageLink,
			},
		},
	}})
}

func updateError(e *events.ApplicationCommandInteractionCreate, message string) {
	updateInteraction(e, discord.MessageUpdate{
		Content: json.Ptr(message),
	})
}

func updateInteraction(e *events.ApplicationCommandInteractionCreate, messageUpdate discord.MessageUpdate) {
	if _, err := e.Client().Rest().UpdateInteractionResponse(e.ApplicationID(), e.Token(), messageUpdate); err != nil {
		logger.Error("error updating interaction: ", err)
	}
}
