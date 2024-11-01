package webserver

import "go.uber.org/zap"

var (
	// Logger to use across the whole webserver.
	Logger *zap.Logger = zap.Must(zap.NewProduction())
)
