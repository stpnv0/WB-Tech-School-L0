package main

import (
	"log/slog"
	"net/http"
	_ "net/http/pprof"
)

func startDebugServer(logger *slog.Logger) {
	go func() {
		logger.Info("pprof debug server started on :6060")
		http.ListenAndServe(":6060", nil)
	}()
}
