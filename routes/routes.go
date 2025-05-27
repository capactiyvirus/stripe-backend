// routes/routes.go
package routes

import (
	"net/http"

	"github.com/capactiyvirus/stripe-backend/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetupRoutes(h *handlers.Handlers) chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)

	// CORS middleware (adjust origins as needed)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*") // Configure this properly
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Health check
	r.Get("/health", handlers.HealthCheck)

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

// SetupRoutesWithAuth sets up routes with authentication middleware
func SetupRoutesWithAuth(h *handlers.Handlers, authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := SetupRoutes(h)

	// Add authentication to admin routes
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(authMiddleware) // Apply auth middleware to all admin routes

		r.Get("/payments", h.GetAllPayments)
		r.Get("/stats", h.GetPaymentStats)
		r.Post("/fulfill/{orderID}", h.FulfillOrder)
		r.Post("/refund/{orderID}", h.RefundOrder)
	})

	return r
}
