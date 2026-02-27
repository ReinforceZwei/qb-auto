package main

import (
	"context"
	"log"
	"os"

	"github.com/ReinforceZwei/qb-auto/clients/animelist"
	quiclient "github.com/ReinforceZwei/qb-auto/clients/qui"
	rsyncclient "github.com/ReinforceZwei/qb-auto/clients/rsync"
	tmdbclient "github.com/ReinforceZwei/qb-auto/clients/tmdb"
	webhookclient "github.com/ReinforceZwei/qb-auto/clients/webhook"
	"github.com/ReinforceZwei/qb-auto/config"
	"github.com/ReinforceZwei/qb-auto/llm"
	"github.com/ReinforceZwei/qb-auto/routes"
	"github.com/ReinforceZwei/qb-auto/workers"
	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"

	_ "github.com/ReinforceZwei/qb-auto/migrations"
)

func main() {
	// Load .env file if present (non-fatal when missing — env vars override JSON config values)
	_ = godotenv.Load()

	cfgPath, err := config.ConfigPath()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if initErr := config.InitConfig(cfgPath); initErr != nil {
			log.Fatal(initErr)
		}
		log.Printf("Config file created at %s — please edit it and restart.", cfgPath)
		os.Exit(0)
	}

	cfg, err := config.LoadFromFile(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	// Inject --http / --https from config into os.Args for the serve subcommand,
	// but only when the flag hasn't already been provided on the CLI.
	injectServeFlag("--http", cfg.HttpAddr)
	injectServeFlag("--https", cfg.HttpsAddr)

	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Only auto migrate when running from go run
		Automigrate: osutils.IsProbablyGoRun(),
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		routes.RegisterTorrentRoutes(se, cfg)

		ctx := context.Background()

		quiClient := quiclient.New(cfg)

		llmClient, err := llm.New(ctx, cfg)
		if err != nil {
			return err
		}

		tmdbClient, err := tmdbclient.New(cfg)
		if err != nil {
			return err
		}

		animeListClient, err := animelist.New(cfg)
		if err != nil {
			return err
		}

		tw := workers.NewTitleWorker(app, cfg, quiClient, llmClient, tmdbClient, animeListClient)
		tw.Register()
		tw.Start(ctx)

		rsyncClient := rsyncclient.NewClient(cfg)
		rw := workers.NewRsyncWorker(app, cfg, quiClient, rsyncClient, animeListClient)
		rw.Register()
		rw.Start(ctx)

		webhookClient := webhookclient.New(cfg)
		nw := workers.NewNotifyWorker(app, cfg, quiClient, webhookClient)
		nw.Register()
		nw.Start(ctx)

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// injectServeFlag appends flag and value to os.Args when the serve subcommand is
// present, the value is non-empty, and the flag has not already been provided.
func injectServeFlag(flag, value string) {
	if value == "" {
		return
	}
	hasServe := false
	flagSet := false
	for _, arg := range os.Args[1:] {
		if arg == "serve" {
			hasServe = true
		}
		if arg == flag || len(arg) > len(flag)+1 && arg[:len(flag)+1] == flag+"=" {
			flagSet = true
		}
	}
	if hasServe && !flagSet {
		os.Args = append(os.Args, flag, value)
	}
}
