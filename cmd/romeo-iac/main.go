package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ctfer-io/romeo/iac"
	"github.com/urfave/cli/v2"
)

var (
	version = "dev"
)

func main() {
	app := cli.App{
		Name:        "Romeo-iac",
		Description: "Romeo belongs to the cloud, not a book.",
		Commands: []*cli.Command{
			{
				Name:   "up",
				Usage:  "Spins up a Romeo instance in the Kubernetes cluster.",
				Action: iac.Up,
			}, {
				Name:    "down",
				Usage:   "Spins down a Romeo instance in the Kubernetes cluster.",
				Aliases: []string{"dn"},
				Action:  iac.Down,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "url",
						Usage:   "The URL to reach the Kubernetes cluster at. Required if coverfile is set.",
						EnvVars: []string{"URL"},
					},
					&cli.StringFlag{
						Name:    "coverfile",
						Usage:   "The coverage file to export Romeo results to in (textfmt).",
						EnvVars: []string{"COVERFILE"},
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "url",
				Usage:   "The URL to reach the Kubernetes cluster at. Required if coverfile is set.",
				EnvVars: []string{"URL"},
			},
			&cli.StringFlag{
				Name:    "coverfile",
				Usage:   "The coverage file to export Romeo results to in (textfmt).",
				EnvVars: []string{"COVERFILE"},
			},
		},
		Action:  run,
		Version: version,
		Authors: []*cli.Author{
			{
				Name:  "Lucas Tesson",
				Email: "lucastesson@protonmail.com",
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		signal.Stop(sigs)
		cancel()
	}()
	go func() {
		select {
		case <-sigs:
			fmt.Println("Signal interruption catched")
			cancel()
			os.Exit(1)
		case <-ctx.Done():
			return
		}
	}()

	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Printf("Root level error: %s", err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	if _, err := os.Stat(iac.StateFile); err != nil {
		return iac.Up(ctx)
	}
	return iac.Down(ctx)
}
