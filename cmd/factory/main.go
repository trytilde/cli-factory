package main

import (
	"context"
	"os"

	"cli-factory/internal/app"
	"cli-factory/internal/cli"
)

func main() {
	registry, err := app.Registry()
	if err != nil {
		panic(err)
	}
	os.Exit(cli.App{Registry: registry}.Run(context.Background(), os.Args[1:]))
}
