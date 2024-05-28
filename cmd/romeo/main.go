package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ctfer-io/romeo"
	"github.com/ctfer-io/romeo/global"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	version = "dev"
)

func main() {
	app := cli.App{
		Name:        "Romeo",
		Description: "O Romeo, Romeo, whatfore art coverages Romeo?",
		Flags: []cli.Flag{
			cli.HelpFlag,
			cli.VersionFlag,
			&cli.StringFlag{
				Name:     "coverdir",
				EnvVars:  []string{"COVERDIR"},
				Required: true,
			},
			&cli.IntFlag{
				Name:    "port",
				EnvVars: []string{"PORT"},
				Value:   8080,
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
			global.Log().Info("signal interruption catched")
			cancel()
			os.Exit(1)
		case <-ctx.Done():
			return
		}
	}()

	if err := app.RunContext(ctx, os.Args); err != nil {
		global.Log().Error("root level error", zap.Error(err))
	}
}

func run(ctx *cli.Context) error {
	romeo.Coverdir = ctx.String("coverdir")

	router := gin.New()
	logger := global.Log()
	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	router.GET("/coverout", romeo.Coverout)

	port := ctx.Int("port")
	logger.Info("api server listening",
		zap.Int("port", port),
	)
	return router.Run(fmt.Sprintf(":%d", port))
}
