package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

// Init initializes the global Zerolog instance.
func Init(environment string) {
	zerolog.TimeFieldFormat = time.RFC3339

	var output = os.Stdout

	if environment == "development" {
		Log = zerolog.New(zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: "15:04:05",
		}).With().
			Timestamp().
			Str("service", "dexta-backend").
			Logger()
	} else {
		Log = zerolog.New(output).With().
			Timestamp().
			Str("service", "dexta-backend").
			Logger()
	}
}

// WithTrace returns a sub-logger tagged with a specific trace ID.
func WithTrace(traceID string) zerolog.Logger {
	return Log.With().Str("trace_id", traceID).Logger()
}

