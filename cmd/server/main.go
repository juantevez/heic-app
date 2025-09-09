// cmd/server/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "github.com/juantevez/heic-app/internal/adapters/http"
	"github.com/juantevez/heic-app/internal/adapters/http/handlers"
	"github.com/juantevez/heic-app/internal/adapters/repositories"
	adapterServices "github.com/juantevez/heic-app/internal/adapters/services"
	"github.com/juantevez/heic-app/internal/config"
	domainServices "github.com/juantevez/heic-app/internal/domain/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting HEIC Photo Processor Service on %s", cfg.GetServerAddress())

	// Initialize database connection
	db, err := repositories.NewPostgresConnection(cfg.GetDatabaseConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Database connection established successfully")

	// Initialize repositories
	photoRepo := repositories.NewPostgresPhotoRepository(db)

	// Initialize services
	exifExtractor := adapterServices.NewExifExtractorService()
	photoService := domainServices.NewPhotoService(photoRepo, exifExtractor)

	// Initialize handlers
	photoHandler := handlers.NewPhotoHandler(photoService)

	// Initialize router
	router := httpAdapter.NewRouter(photoHandler)
	routes := router.SetupRoutes()

	// Configure HTTP server
	server := &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      routes,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", cfg.GetServerAddress())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	log.Println("Server started successfully")
	log.Println("Available endpoints:")
	log.Println("  POST   /api/v1/photos")
	log.Println("  GET    /api/v1/photos/{id}")
	log.Println("  GET    /api/v1/photos/{id}/download")
	log.Println("  GET    /api/v1/bikes/{bike_id}/photos")
	log.Println("  GET    /api/v1/health")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}

// Application struct for dependency injection (alternative approach)
type Application struct {
	config       *config.Config
	photoService *domainServices.PhotoServiceImpl
	photoHandler *handlers.PhotoHandler
	server       *http.Server
}

func NewApplication() (*Application, error) {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	// Initialize database
	db, err := repositories.NewPostgresConnection(cfg.GetDatabaseConnectionString())
	if err != nil {
		return nil, err
	}

	// Initialize dependencies
	photoRepo := repositories.NewPostgresPhotoRepository(db)
	exifExtractor := adapterServices.NewExifExtractorService()
	photoService := domainServices.NewPhotoService(photoRepo, exifExtractor)
	photoHandler := handlers.NewPhotoHandler(photoService)

	// Setup router
	router := httpAdapter.NewRouter(photoHandler)
	routes := router.SetupRoutes()

	// Configure server
	server := &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      routes,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Application{
		config:       cfg,
		photoService: photoService,
		photoHandler: photoHandler,
		server:       server,
	}, nil
}

func (app *Application) Run() error {
	return app.server.ListenAndServe()
}

func (app *Application) Shutdown(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}
