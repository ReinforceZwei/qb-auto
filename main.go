package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

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
	// install subcommand needs no config — short-circuit before config loading.
	if len(os.Args) > 1 && os.Args[1] == "install" {
		app := pocketbase.New()
		app.RootCmd.AddCommand(newInstallCmd())
		if err := app.Start(); err != nil {
			log.Fatal(err)
		}
		return
	}

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

	// Store pb_data next to the config file so data follows the same XDG location.
	pbDataDir := filepath.Join(filepath.Dir(cfgPath), "pb_data")

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: pbDataDir,
	})

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Only auto migrate when running from go run
		Automigrate: osutils.IsProbablyGoRun(),
	})

	// Apply HTTP listen address from config before the listener is created.
	// Priority 999 ensures this runs before the PocketBase internal finalizer.
	if cfg.HttpAddr != "" {
		app.OnServe().BindFunc(func(se *core.ServeEvent) error {
			se.Server.Addr = cfg.HttpAddr
			return se.Next()
		})
	}

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

	app.RootCmd.AddCommand(newInstallCmd())

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

