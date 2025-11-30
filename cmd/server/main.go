package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var (
	logLevel slog.LevelVar
)

func main() {
	logLevel.Set(slog.LevelInfo)
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     &logLevel,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					src.File = filepath.Base(src.File)
					a.Value = slog.AnyValue(src)
				}
			}
			return a
		},
	})
	slog.SetDefault(slog.New(handler))

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	rc, err := NewRootCmd()
	if err != nil {
		slog.Error("start gophkeeper server", "error", err)
		os.Exit(1)
	}

	if err := rc.ExecuteContext(ctx); err != nil {
		slog.Error("start gophkeeper server", "error", err)
		os.Exit(1)
	}
}
