package iac

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func FetchCoverfile(ctx context.Context, url, coverfile string) error {
	client := &http.Client{}

	// Fetch coverout
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/coverout", url), nil)
	req.Header.Set("User-Agent", "romeo-iac")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Unmarshal JSON response
	var cov Coverout
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&cov); err != nil {
		return err
	}

	// Decode base 64
	b, err := base64.StdEncoding.DecodeString(cov.Merged)
	if err != nil {
		return err
	}
	buf := bytes.NewReader(b)

	r, err := zip.NewReader(buf, buf.Size())
	if err != nil {
		return err
	}
	dir := filepath.Join(os.TempDir(), "romeo")
	for _, f := range r.File {
		if err := unzip(dir, f); err != nil {
			return err
		}
	}

	// Merge coverages in textfmt
	cmd := exec.Command("go", "tool", "covdata", "textfmt", "-i", dir, "-o", coverfile)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func unzip(root string, f *zip.File) error {
	fpath := filepath.Join(root, f.Name)
	file, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer file.Close()

	fileInArchive, err := f.Open()
	if err != nil {
		return err
	}
	defer fileInArchive.Close()

	if _, err := io.Copy(file, fileInArchive); err != nil {
		return err
	}

	return nil
}

type Coverout struct {
	Merged string `json:"merged"`
}
