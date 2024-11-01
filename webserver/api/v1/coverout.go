package apiv1

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ctfer-io/romeo/webserver"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CoveroutResponse is the response to a GET /coverout call
type CoveroutResponse struct {
	Merged string `json:"merged"`
}

var (
	Coverdir = ""
)

func Coverout(ctx *gin.Context) {
	// Create temporary directory
	tmpDir, rm := newTmpDir()
	if rm == nil {
		return
	}
	defer rm()

	// Merge files
	// TODO bind to Go's internals rather than executing it (smaller Docker images and avoid CLI flags injections)
	cmd := exec.Command("go", "tool", "covdata", "merge", "-i="+Coverdir, "-o="+tmpDir)
	if err := cmd.Run(); err != nil {
		internalErr(ctx, "merge returned non-zero status code")
		return
	}

	merged, err := Encode(tmpDir)
	if err != nil {
		internalErr(ctx, err.Error())
		return
	}

	// Encode b64 and serve it
	ctx.JSON(http.StatusOK, CoveroutResponse{
		Merged: merged,
	})
}

func newTmpDir() (string, func()) {
	// Generate random name
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	tmpDir := filepath.Join(os.TempDir(), hex.EncodeToString(b))

	// Create directory
	if err := os.Mkdir(tmpDir, os.ModePerm); err != nil {
		webserver.Logger.Error("creating temporary directory failed",
			zap.String("directory", tmpDir),
			zap.Error(err),
		)
		return "", nil
	}

	return tmpDir, func() {
		// Delete directory
		if err := os.Remove(tmpDir); err != nil {
			webserver.Logger.Error("deleting temporary directory failed",
				zap.String("directory", tmpDir),
				zap.Error(err),
			)
		}
	}
}

func internalErr(ctx *gin.Context, err string) {
	webserver.Logger.Error("internal error", zap.String("err", err))
	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": err,
	})
}
