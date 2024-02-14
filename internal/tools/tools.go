package tools

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetGlobalLogLevel(level string) {
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Log().Msg("Set debug logLevel")
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Log().Msg("Set info logLevel")
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		log.Log().Msg("Set error logLevel")
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
		log.Log().Msg("Set fatal logLevel")
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Log().Msgf("Invalid log level %s, defaulting to info level", level)
	}
}
