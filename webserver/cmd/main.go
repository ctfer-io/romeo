package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ctfer-io/romeo/webserver"
	apiv1 "github.com/ctfer-io/romeo/webserver/api/v1"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

func main() {
	app := cli.App{
		Name:        "Romeo - Webserver",
		Description: "O Romeo, Romeo, whatfore art coverages Romeo?",
		Flags: []cli.Flag{
			cli.HelpFlag,
			cli.VersionFlag,
			&cli.StringFlag{
				Name:        "coverdir",
				EnvVars:     []string{"COVERDIR"},
				Destination: &apiv1.Coverdir,
			},
			&cli.IntFlag{
				Name:    "port",
				EnvVars: []string{"PORT"},
				Value:   8080,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "download",
				Usage: "Download the Romeo data from an environment, after running your tests.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "server",
						Usage:    "Server URL to reach out the Romeo environment.",
						Required: true,
						EnvVars:  []string{"SERVER"},
					},
					&cli.StringFlag{
						Name:     "directory",
						Usage:    "Directory to export the coverages data (defaults to \"coverout\").",
						Required: true,
						EnvVars:  []string{"DIRECTORY"},
					},
				},
				Action: download,
			},
		},
		Action: run,
		Authors: []*cli.Author{
			{
				Name:  "Lucas Tesson - PandatiX",
				Email: "lucastesson@protonmail.com",
			},
		},
		Version: version,
		Metadata: map[string]any{
			"version": version,
			"commit":  commit,
			"date":    date,
			"builtBy": builtBy,
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
			webserver.Logger.Info("signal interruption catched")
			cancel()
			os.Exit(1)
		case <-ctx.Done():
			return
		}
	}()

	if err := app.RunContext(ctx, os.Args); err != nil {
		webserver.Logger.Error("root level error", zap.Error(err))
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(ginzap.Ginzap(webserver.Logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(webserver.Logger, true))

	apiv1g := router.Group("/api/v1")
	apiv1g.GET("/coverout", apiv1.Coverout)

	port := ctx.Int("port")
	webserver.Logger.Info("api server listening",
		zap.Int("port", port),
	)
	return router.Run(fmt.Sprintf(":%d", port))
}

func download(ctx *cli.Context) error {
	// Download coverages
	server := ctx.String("server")
	fmt.Printf("Downloading coverages from %s...\n", server)
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/coverout", server), nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Decode them
	resp := &apiv1.CoveroutResponse{}
	if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
		return err
	}

	// Export to filesystem
	cd := ctx.String("directory")
	fmt.Printf("Exporting coverages to %s\n", cd)
	if err := apiv1.Decode(resp.Merged, cd); err != nil {
		return errors.Wrap(err, "decoding coverages")
	}

	// Write coverdir as an output
	return webserver.Output("directory", cd)
}
