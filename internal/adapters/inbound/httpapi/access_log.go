package httpapi

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// slowRequest is how long a request may take before it is logged even when it succeeded.
const slowRequest = time.Second

// accessLog logs the requests worth reading: every change (not GET/HEAD/OPTIONS), every
// failure (status 400 and up) and every slow request. Successful reads and session heartbeats
// are left out: UIs and extensions poll every few seconds and made up most of the log (65 %
// were heartbeats) while hiding what happened.
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		took := time.Since(start)
		if !worthLogging(r, status, took) {
			return
		}
		log.Printf("%s %s -> %d %dB in %s from %s", r.Method, r.URL.RequestURI(), status, ww.BytesWritten(), took.Round(time.Microsecond), r.RemoteAddr)
	})
}

// worthLogging decides whether accessLog writes a line for this request.
func worthLogging(r *http.Request, status int, took time.Duration) bool {
	if status >= http.StatusBadRequest || took >= slowRequest {
		return true
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	return !strings.HasSuffix(r.URL.Path, "/heartbeat")
}
