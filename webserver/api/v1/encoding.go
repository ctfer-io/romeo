package apiv1

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const (
	maxSize = 1 << 30 // 1Gb
)

// Encode consumes a source  directory, zip its content and encoded
// base 64.
func Encode(src string) (string, error) {
	// Create a zip-buffer
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)

	// Walk in filesystem to zip files
	if err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		file, err := os.Open(path) //nolint:gosec //#gosec G122 -- FP, the path cannot be controlled by an attacker
		if err != nil {
			return err
		}

		// Remove coverdir from file in zip (avoid nested directories)
		npath := strings.TrimPrefix(path, src+"/")
		f, err := w.Create(npath)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		file.Close()

		return err
	}); err != nil {
		return "", err
	}
	w.Close()

	// Encode base 64
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// Decode consumes a buffer, decodes base 64 and unzip it to a given
// destination.
func Decode(buf string, dst string) error {
	// Decode base 64
	raw, err := base64.StdEncoding.DecodeString(buf)
	if err != nil {
		return errors.Wrap(err, "base64 decoding")
	}

	// Safely decompress the archive (borrowed from Chall-Manager)
	r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return errors.Wrap(err, "base64 decoded invalid zip archive")
	}

	dec := NewDecompressor(&Options{
		MaxSize: maxSize,
	})
	_, err = dec.Unzip(r, dst)
	return err
}
