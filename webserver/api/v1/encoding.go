package apiv1

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// Encode consumes a source  directory, zip its content and encoded
// base 64.
func Encode(src string) (string, error) {
	// Create a zip-buffer
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)

	// Walk in filesystem to zip files
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
		npath := strings.TrimPrefix(path, src+"/")
		f, err := w.Create(npath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(f, file); err != nil {
			return err
		}

		return nil
	}
	if err := filepath.Walk(src, walker); err != nil {
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

	// Unzip content into it
	r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return errors.Wrap(err, "base64 decoded invalid zip archive")
	}
	for _, f := range r.File {
		filePath := filepath.Join(dst, f.Name)
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

	return nil
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
