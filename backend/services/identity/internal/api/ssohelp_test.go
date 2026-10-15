package api

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// chiStub records what was mounted without needing a real router.
type chiStub struct {
	chi.Router
	paths []string
}

func (c *chiStub) Get(pattern string, _ http.HandlerFunc) {
	c.paths = append(c.paths, pattern)
}
