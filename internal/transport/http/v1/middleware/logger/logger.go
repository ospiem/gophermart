package logger

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/go-chi/chi/v5/middleware"
)

func RequestLogger(log zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			uri := r.RequestURI
			method := r.Method

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				log.Info().
					Str("URI", uri).
					Str("Method", method).
					Str("Duration", time.Since(start).String()).
					Int("Bytes", ww.BytesWritten()).
					Int("Status", ww.Status()).
					Msg("")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
