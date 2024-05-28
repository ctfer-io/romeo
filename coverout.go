package romeo

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ctfer-io/romeo/global"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	Coverdir string
)

func Coverout(ctx *gin.Context) {
	// Create temporary directory
	tmpDir, rm := newTmpDir()
	if rm == nil {
		return
	}
	defer rm()

	// Merge files
	cmd := exec.Command("go", "tool", "covdata", "merge", "-i="+Coverdir, "-o="+tmpDir)
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
	global.Log().Error("internal error", zap.String("err", err))
	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": err,
	})
}

func newTmpDir() (string, func()) {
	logger := global.Log()

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
