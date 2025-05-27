// routes/payment_routes.go
package routes

import (
	"github.com/capactiyvirus/stripe-backend/handlers"
	"github.com/go-chi/chi/v5"
)

func SetupPaymentRoutes(r chi.Router, h *handlers.Handlers) {
	r.Route("/api/payments", func(r chi.Router) {
		// Payment creation routes
		r.Post("/create-intent", h.CreatePaymentIntent)
		r.Post("/create-checkout", h.CreateCheckoutSession)
		r.Post("/create-order", h.CreateOrder) // New: Create order with tracking

		// Payment verification and status
		r.Get("/verify/{id}", h.VerifyPayment)
		r.Get("/status/{orderID}", h.GetPaymentStatus) // New: Get payment status by order ID
		r.Get("/order/{orderID}", h.GetOrderDetails)   // New: Get full order details

		// Payment tracking
		r.Get("/track/{trackingID}", h.TrackPayment)      // New: Track payment by tracking ID
		r.Get("/customer/{email}", h.GetCustomerPayments) // New: Get customer payment history

		// Admin routes (you may want to add auth middleware)
		r.Get("/all", h.GetAllPayments)    // New: Get all payments (admin)
		r.Get("/stats", h.GetPaymentStats) // New: Get payment statistics

		// Webhook handler
		r.Post("/webhook", h.HandleStripeWebhook)

		// Order fulfillment
		r.Post("/fulfill/{orderID}", h.FulfillOrder) // New: Mark order as fulfilled
		r.Post("/refund/{orderID}", h.RefundOrder)   // New: Process refund
	})
}
