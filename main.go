package main

import (
	"context"
	"log"

	"github.com/ReinforceZwei/qb-auto/clients/animelist"
	quiclient "github.com/ReinforceZwei/qb-auto/clients/qui"
	tmdbclient "github.com/ReinforceZwei/qb-auto/clients/tmdb"
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
	// Load .env file if present (non-fatal when missing — production uses real env vars)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

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

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
