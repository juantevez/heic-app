// internal/adapters/http/router.go
package http

import (
	"log"
	"net/http"
	"time"

	"github.com/juantevez/heic-app/internal/adapters/http/handlers"

	"github.com/gorilla/mux"
)

type Router struct {
	photoHandler *handlers.PhotoHandler
}

func NewRouter(photoHandler *handlers.PhotoHandler) *Router {
	return &Router{
		photoHandler: photoHandler,
	}
}

func (rt *Router) SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Add middleware
	r.Use(rt.loggingMiddleware)
	r.Use(rt.corsMiddleware)
	r.Use(rt.contentTypeMiddleware)

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Photo routes
	api.HandleFunc("/photos", rt.photoHandler.UploadPhoto).Methods("POST")
	api.HandleFunc("/photos/{id}", rt.photoHandler.GetPhoto).Methods("GET")
	api.HandleFunc("/photos/{id}/download", rt.photoHandler.DownloadPhoto).Methods("GET")
	api.HandleFunc("/bikes/{bike_id}/photos", rt.photoHandler.GetPhotosByBike).Methods("GET")

	// Health check
	api.HandleFunc("/health", rt.photoHandler.HealthCheck).Methods("GET")

	return r
}

// Middleware functions
func (rt *Router) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		log.Printf("[%s] %s %s - %d - %v",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			wrapped.statusCode,
			duration,
		)
	})
}

func (rt *Router) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rt *Router) contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip content-type check for multipart uploads
		if r.Header.Get("Content-Type") != "" &&
			r.Method == "POST" &&
			r.URL.Path == "/api/v1/photos" {
			next.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Response writer wrapper to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
