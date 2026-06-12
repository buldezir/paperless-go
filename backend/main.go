package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"
	"paperless-go/backend/internal/hooks"
	"paperless-go/backend/internal/worker"

	_ "paperless-go/backend/migrations"
)

func main() {
	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: osutils.IsProbablyGoRun(),
	})

	hooks.Register(app)
	worker.Start(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
