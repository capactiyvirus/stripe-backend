package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/stripe/stripe-go/v82"
	
	"github.com/yourusername/stripe-api/config"
	"github.com/yourusername/stripe-api/routes"
)

func main() {
	// Load configuration
	cfg := config.Load()
	
	// Initialize Stripe
	stripe.Key = cfg.StripeSecretKey
	
	// Create router
	r := chi.NewRouter()
	
	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	
	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CorsAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	
	// Setup routes
	routes.SetupRoutes(r, cfg)
	
	// Start server
	log.Printf("Server starting on port %s in %s mode", cfg.Port, cfg.Environment)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}