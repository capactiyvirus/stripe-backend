// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/capactiyvirus/stripe-backend/config"
	"github.com/capactiyvirus/stripe-backend/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stripe/stripe-go/v82"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Set Stripe API key
	stripe.Key = cfg.StripeSecretKey

	// Create handlers with payment store
	h := handlers.NewHandlers(cfg)

	// Setup routes
	r := setupRouter(cfg, h)

	// Start server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", cfg.Port)
		log.Printf("Environment: %s", cfg.Environment)
		log.Printf("CORS allowed origins: %v", cfg.CorsAllowedOrigins)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Attempt graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(cfg *config.Config, h *handlers.Handlers) chi.Router {
	r := chi.NewRouter()

	// Basic middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	r.Use(corsMiddleware(cfg.CorsAllowedOrigins))

	// Security headers middleware
	r.Use(securityMiddleware)

	// Health check endpoint - Fixed to use handler method
	r.Get("/health", h.HealthCheck)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok", "service": "payment-api", "version": "1.0.0"}`)
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Payment routes with enhanced tracking
		r.Route("/payments", func(r chi.Router) {
			// Payment creation routes
			r.Post("/create-intent", h.CreatePaymentIntent)     // Legacy support
			r.Post("/create-checkout", h.CreateCheckoutSession) // Legacy support
			r.Post("/create-order", h.CreateOrder)              // New: Create order with tracking

			// Payment verification and status
			r.Get("/verify/{id}", h.VerifyPayment)         // Legacy support
			r.Get("/status/{orderID}", h.GetPaymentStatus) // New: Get payment status by order ID
			r.Get("/order/{orderID}", h.GetOrderDetails)   // New: Get full order details

			// Payment tracking
			r.Get("/track/{trackingID}", h.TrackPayment)      // New: Track payment by tracking ID
			r.Get("/customer/{email}", h.GetCustomerPayments) // New: Get customer payment history

			// Admin routes (consider adding authentication middleware)
			r.Get("/all", h.GetAllPayments)    // New: Get all payments (admin)
			r.Get("/stats", h.GetPaymentStats) // New: Get payment statistics

			// Order fulfillment
			r.Post("/fulfill/{orderID}", h.FulfillOrder) // New: Mark order as fulfilled
			r.Post("/refund/{orderID}", h.RefundOrder)   // New: Process refund

			// Webhook handler
			r.Post("/webhook", h.HandleStripeWebhook) // Enhanced webhook handling
		})

		// Product routes (for integration with your Next.js app)
		r.Route("/products", func(r chi.Router) {
			r.Get("/", h.ListProducts)   // List available products
			r.Get("/{id}", h.GetProduct) // Get single product details
		})
	})

	return r
}

// corsMiddleware handles CORS headers
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// securityMiddleware adds security headers
func securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Only set HSTS in production
		if os.Getenv("ENVIRONMENT") == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
