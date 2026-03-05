package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ReinforceZwei/qb-auto/clients/animelist"
	braveclient "github.com/ReinforceZwei/qb-auto/clients/brave"
	quiclient "github.com/ReinforceZwei/qb-auto/clients/qui"
	rsyncclient "github.com/ReinforceZwei/qb-auto/clients/rsync"
	tmdbclient "github.com/ReinforceZwei/qb-auto/clients/tmdb"
	webhookclient "github.com/ReinforceZwei/qb-auto/clients/webhook"
	wikiclient "github.com/ReinforceZwei/qb-auto/clients/wikipedia"
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

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Load .env file if present (non-fatal when missing — env vars override JSON config values)
	_ = godotenv.Load()

	// Get config path to build data dir, actual config load happens on serve command
	cfgPath, err := config.ConfigPath()
	if err != nil {
		log.Fatal(err)
	}

	// Store pb_data next to the config file so data follows the same XDG location.
	pbDataDir := filepath.Join(filepath.Dir(cfgPath), "pb_data")

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: pbDataDir,
	})

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Only auto migrate when running from go run
		Automigrate: osutils.IsProbablyGoRun(),
	})

	app.RootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	app.RootCmd.SetVersionTemplate("{{.Name}} version {{.Version}}\n")

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Create init config if not exist then exit
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
		if cfg.HttpAddr != "" {
			se.Server.Addr = cfg.HttpAddr
		}

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

		var braveClient *braveclient.Client
		if cfg.BraveAPIKey != "" {
			braveClient = braveclient.New(cfg.BraveAPIKey)
		}
		wikiClient := wikiclient.New()

		routes.RegisterAnimeTitleRoutes(se, llmClient, tmdbClient, animeListClient, braveClient, wikiClient)

		tw := workers.NewTitleWorker(app, cfg, quiClient, llmClient, tmdbClient, animeListClient, braveClient, wikiClient)
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

		// Re-enqueue any jobs that were interrupted by a previous crash or restart.
		tw.Recover()
		rw.Recover()
		nw.Recover()

		return se.Next()
	})

	app.RootCmd.AddCommand(newInstallCmd())
	app.RootCmd.AddCommand(newUpdateCmd())

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
