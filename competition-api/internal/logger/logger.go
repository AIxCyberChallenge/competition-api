package logger

import (
	"log/slog"
	"os"

	slogotel "github.com/remychantenay/slog-otel"
)

var LogLevel = new(slog.LevelVar)

var jsonHandler = slog.NewJSONHandler(
	os.Stderr,
	&slog.HandlerOptions{AddSource: true, Level: LogLevel},
)
var sloghandler = slogotel.NewOtelHandler(slogotel.WithNoTraceEvents(true))
var Handler = sloghandler(jsonHandler)
var Logger = slog.New(Handler)

func InitSlog() {
	slog.SetDefault(Logger)
	LogLevel.Set(slog.LevelDebug)
}
