package webserver

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func Output(key, dir string) error {
	// Open GitHub output file
	//nolint:lll // the line is long for gosec FP description
	f, err := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600) //nolint:gosec //#gosec G703 -- FP, this is intended
	if err != nil {
		return errors.Wrap(err, "opening output file")
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			Logger.Error("closing GitHub output file", zap.Error(err))
		}
	}(f)

	// Write and ensure it went fine
	if _, err = fmt.Fprintf(f, "%s=%s\n", key, dir); err != nil {
		return errors.Wrapf(err, "writing %s output", key)
	}

	return nil
}
