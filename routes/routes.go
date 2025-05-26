package routes

import (
	"github.com/go-chi/chi/v5"
	
	"github.com/capactiyvirus/stripe-backend/config"
	"github.com/capactiyvirus/stripe-backend/handlers"
)

func SetupRoutes(r chi.Router, cfg *config.Config) {
	// Create handler instance
	h := handlers.NewHandlers(cfg)
	
	// Health check - use the method on the handler instance
	r.Get("/health", h.HealthCheck)
	
	// API routes
	r.Route("/api", func(r chi.Router) {
		// Stripe payment routes
		r.Route("/payments", func(r chi.Router) {
			r.Post("/create-intent", h.CreatePaymentIntent)
			r.Post("/create-checkout", h.CreateCheckoutSession)
			r.Get("/verify/{id}", h.VerifyPayment)
		})
		
		// Stripe products routes
		r.Route("/products", func(r chi.Router) {
			r.Get("/", h.ListProducts)
			r.Get("/{id}", h.GetProduct)
		})
		
		// Webhook handler
		r.Post("/webhook", h.HandleStripeWebhook)
	})
}