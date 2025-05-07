package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/yourusername/stripe-api/handlers"
)

func SetupRoutes(r chi.Router) {
	// Health check
	r.Get("/health", handlers.HealthCheck)
	
	// API routes
	r.Route("/api", func(r chi.Router) {
		// Stripe payment routes
		r.Route("/payments", func(r chi.Router) {
			r.Post("/create-intent", handlers.CreatePaymentIntent)
			r.Post("/create-checkout", handlers.CreateCheckoutSession)
			r.Get("/verify/{id}", handlers.VerifyPayment)
		})
		
		// Stripe products routes
		r.Route("/products", func(r chi.Router) {
			r.Get("/", handlers.ListProducts)
			r.Get("/{id}", handlers.GetProduct)
		})
		
		// Webhook handler
		r.Post("/webhook", handlers.HandleStripeWebhook)
	})
}