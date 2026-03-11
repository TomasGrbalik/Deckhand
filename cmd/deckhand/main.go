package main

import (
	"os"

	"github.com/TomasGrbalik/deckhand/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
