package mediasoup

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
)

var (
	// default zerolog
	zl = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.TimeFormat = "2006-01-02 15:04:05.000000"
	})).With().Caller().Timestamp().Logger().Level(zerolog.InfoLevel)

	// NewLogger defines function to create logger instance.
	NewLogger = func(scope string) logr.Logger {
		return zerologr.New(&zl).WithName(scope)
	}
)
