package main

import (
	"os"

	"github.com/ospiem/gophermart/internal/server"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	if err := server.Run(logger); err != nil {
		logger.Fatal().Err(err)
	}
	logger.Info().Msg("Graceful shutdown completed successfully. All connections closed, and resources released.")
}
