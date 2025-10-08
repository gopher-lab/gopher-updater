package xlog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var (
	logger  *slog.Logger
	leveler *slog.LevelVar
)

func init() {
	leveler = new(slog.LevelVar)
	level, err := ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = slog.LevelDebug
	}
	leveler.Set(level)

	var opts = &slog.HandlerOptions{
		Level: leveler,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger = slog.New(handler)
}

func SetLevel(level slog.Level) {
	leveler.Set(level)
	Warn("Log level set", "level", leveler.Level().String())
}

func ParseLevel(lvl string) (slog.Level, error) {
	l, err := strconv.Atoi(lvl)
	if err == nil {
		return slog.Level(l), nil
	}
	switch strings.ToLower(lvl) {
	case "error":
		return slog.LevelError, nil
	case "warning", "warn":
		return slog.LevelWarn, nil
	case "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	}
	return slog.Level(0), fmt.Errorf("unknown log level: %s", lvl)
}

func _log(level slog.Level, msg string, args ...any) {
	_, f, l, _ := runtime.Caller(2)
	group := slog.Group(
		"source",
		slog.Attr{
			Key:   "filename",
			Value: slog.AnyValue(f),
		},
		slog.Attr{
			Key:   "lineno",
			Value: slog.AnyValue(l),
		},
	)
	args = append(args, group)
	logger.Log(context.Background(), level, msg, args...)
}

func Info(msg string, args ...any) {
	_log(slog.LevelInfo, msg, args...)
}

func Debug(msg string, args ...any) {
	_log(slog.LevelDebug, msg, args...)
}

func Error(msg string, args ...any) {
	_log(slog.LevelError, msg, args...)
}

func Warn(msg string, args ...any) {
	_log(slog.LevelWarn, msg, args...)
}

func Fatal(msg string, args ...any) {
	_log(slog.LevelError, msg, args...)
	os.Exit(1)
}
