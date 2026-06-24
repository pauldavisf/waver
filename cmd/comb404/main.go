package main

import (
	"context"
	"elon/waver/internal/app"
	"elon/waver/internal/ui"
	"fmt"
	"os"
	"os/signal"
	"strings"
)

func main() {
	args := os.Args

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if len(args) <= 1 {
		err := ui.Run(ctx, "127.0.0.1:4040", true)
		if err != nil {
			panic(err)
		}

		return
	}

	if strings.ToLower(args[1]) == "comb" {
		if len(args) <= 2 {
			panic("no pattern path in args")
		}

		_, err := app.Combine(ctx, args[2], "out")
		if err != nil {
			panic(err)
		}

		return
	}

	if strings.ToLower(args[1]) == "ui" {
		addr := "127.0.0.1:4040"
		open := true

		for _, arg := range args[2:] {
			if arg == "--no-open" {
				open = false
				continue
			}

			addr = arg
		}

		err := ui.Run(ctx, addr, open)
		if err != nil {
			panic(err)
		}

		return
	}

	panic(fmt.Sprintf("unknown command: %s", args[1]))
}
