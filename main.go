package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
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

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
