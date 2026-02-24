package main

import (
	"log"

	"github.com/ReinforceZwei/qb-auto/routes"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"

	_ "github.com/ReinforceZwei/qb-auto/migrations"
)

func main() {
	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Only auto migrate when running from go run
		Automigrate: osutils.IsProbablyGoRun(),
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		routes.RegisterTorrentRoutes(se)
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
