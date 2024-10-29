package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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

	logger = zap.Must(zap.NewProduction())
)

func main() {
	app := cli.App{
		Name:        "Romeo - Webserver",
		Description: "O Romeo, Romeo, whatfore art coverages Romeo?",
		Flags: []cli.Flag{
			cli.HelpFlag,
			cli.VersionFlag,
			&cli.StringFlag{
				Name:    "coverdir",
				EnvVars: []string{"COVERDIR"},
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
			logger.Info("signal interruption catched")
			cancel()
			os.Exit(1)
		case <-ctx.Done():
			return
		}
	}()

	if err := app.RunContext(ctx, os.Args); err != nil {
		logger.Error("root level error", zap.Error(err))
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	coverdir = ctx.String("coverdir")

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	router.GET("/coverout", coverout)

	port := ctx.Int("port")
	logger.Info("api server listening",
		zap.Int("port", port),
	)
	return router.Run(fmt.Sprintf(":%d", port))
}

// region API

type CoveroutResponse struct {
	Merged string `json:"merged"`
}

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
	// TODO bind to Go's internals rather than executing it (smaller Docker images and avoid CLI flags injections)
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

		// Remove coverdir from file in zip (avoid nested directories)
		npath := strings.TrimPrefix(path, tmpDir+"/")
		f, err := w.Create(npath)
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
	ctx.JSON(http.StatusOK, CoveroutResponse{
		Merged: base64.StdEncoding.EncodeToString(buf.Bytes()),
	})
}

func internalErr(ctx *gin.Context, err string) {
	logger.Error("internal error", zap.String("err", err))
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
		logger.Error("creating temporary directory failed",
			zap.String("directory", tmpDir),
			zap.Error(err),
		)
		return "", nil
	}

	return tmpDir, func() {
		// Delete directory
		if err := os.Remove(tmpDir); err != nil {
			logger.Error("deleting temporary directory failed",
				zap.String("directory", tmpDir),
				zap.Error(err),
			)
		}
	}
}

// region download

func download(ctx *cli.Context) error {
	// Download coverages
	fmt.Println("Downloading coverages...")
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/coverout", ctx.String("server")), nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// Unmarshal them
	resp := &CoveroutResponse{}
	if err := json.Unmarshal(b, resp); err != nil {
		return errors.Wrapf(err, "unmarshalling error of body %s", b)
	}

	// Decode base64
	raw, err := base64.StdEncoding.DecodeString(resp.Merged)
	if err != nil {
		return errors.Wrap(err, "base64 decoding")
	}

	// Unzip content into it
	r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return errors.Wrap(err, "base64 decoded invalid zip archive")
	}
	cd := ctx.String("directory")
	fmt.Printf("Exporting coverages to %s\n", cd)
	for _, f := range r.File {
		filePath := filepath.Join(cd, f.Name)
		if f.FileInfo().IsDir() {
			continue
		}

		// If the file is in a sub-directory, create it
		dir := filepath.Dir(filePath)
		if _, err := os.Stat(dir); err != nil {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return err
			}
		}

		// Create and write the file
		if err := copyTo(filePath, f); err != nil {
			return err
		}
	}

	// Write coverdir as an output
	return outputDirectory(cd)
}

func copyTo(filePath string, f *zip.File) error {
	outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	defer outFile.Close()

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if _, err := io.Copy(outFile, rc); err != nil {
		return err
	}
	return nil
}

func outputDirectory(dir string) error {
	f, err := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		return errors.Wrap(err, "opening output file")
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "directory=%s\n", dir)
	return errors.Wrap(err, "writing directory output")
}
