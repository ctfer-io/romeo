package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
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
			Log().Info("signal interruption catched")
			cancel()
			os.Exit(1)
		case <-ctx.Done():
			return
		}
	}()

	if err := app.RunContext(ctx, os.Args); err != nil {
		Log().Error("root level error", zap.Error(err))
	}
}

func run(ctx *cli.Context) error {
	coverdir = ctx.String("coverdir")

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	logger := Log()
	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	router.GET("/coverout", coverout)

	port := ctx.Int("port")
	logger.Info("api server listening",
		zap.Int("port", port),
	)
	return router.Run(fmt.Sprintf(":%d", port))
}

// region log

var (
	logger  *zap.Logger
	logOnce sync.Once
)

func Log() *zap.Logger {
	logOnce.Do(func() {
		logger, _ = zap.NewProduction()
	})
	return logger
}

// region API

var (
	coverdir = ""
)

func coverout(ctx *gin.Context) {
	// Create temporary directory
	tmpDir, rm := newTmpDir()
	if rm == nil {
		return
	}
	defer rm()

	// Merge files
	cmd := exec.Command("go", "tool", "covdata", "merge", "-i="+coverdir, "-o="+tmpDir)
	if err := cmd.Run(); err != nil {
		internalErr(ctx, "merge returned non-zero status code")
		return
	}

	// Fetch merged coverage file in input coverage, and zip it
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		f, err := w.Create(path)
		if err != nil {
			return err
		}

		if _, err := io.Copy(f, file); err != nil {
			return err
		}

		return nil
	}
	if err := filepath.Walk(tmpDir, walker); err != nil {
		internalErr(ctx, fmt.Sprintf("zip failed: %s", err))
		return
	}
	w.Close()

	// Encode b64 and serve it
	ctx.JSON(http.StatusOK, gin.H{
		"merged": base64.StdEncoding.EncodeToString(buf.Bytes()),
	})
}

func internalErr(ctx *gin.Context, err string) {
	Log().Error("internal error", zap.String("err", err))
	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": err,
	})
}

func newTmpDir() (string, func()) {
	// Generate random name
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	tmpDir := filepath.Join(os.TempDir(), hex.EncodeToString(b))

	// Create directory
	if err := os.Mkdir(tmpDir, os.ModePerm); err != nil {
		Log().Error("creating temporary directory failed",
			zap.String("directory", tmpDir),
			zap.Error(err),
		)
		return "", nil
	}

	return tmpDir, func() {
		// Delete directory
		if err := os.Remove(tmpDir); err != nil {
			Log().Error("deleting temporary directory failed",
				zap.String("directory", tmpDir),
				zap.Error(err),
			)
		}
	}
}
