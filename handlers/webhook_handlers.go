// handlers/webhook_handlers.go
package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/capactiyvirus/stripe-backend/models"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

// HandleStripeWebhook handles Stripe webhook events with enhanced tracking
func (h *Handlers) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		respondWithError(w, http.StatusServiceUnavailable, "Error reading request body")
		return
	}

	// Verify webhook signature
	endpointSecret := h.config.StripeWebhookSecret
	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		respondWithError(w, http.StatusBadRequest, "Webhook signature verification failed")
		return
	}

	// Handle the event
	switch event.Type {
	case "payment_intent.succeeded":
		h.handlePaymentIntentSucceeded(event)
	case "payment_intent.payment_failed":
		h.handlePaymentIntentFailed(event)
	case "payment_intent.canceled":
		h.handlePaymentIntentCanceled(event)
	case "checkout.session.completed":
		h.handleCheckoutSessionCompleted(event)
	case "invoice.payment_succeeded":
		h.handleInvoicePaymentSucceeded(event)
	case "charge.dispute.created":
		h.handleChargeDisputeCreated(event)
	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handlePaymentIntentSucceeded processes successful payment intents
func (h *Handlers) handlePaymentIntentSucceeded(event stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	err := json.Unmarshal(event.Data.Raw, &paymentIntent)
	if err != nil {
		log.Printf("Error parsing payment_intent.succeeded: %v", err)
		return
	}

	log.Printf("Payment succeeded: %s", paymentIntent.ID)

	// Find the order by payment intent ID
	orderID := h.findOrderByPaymentIntentID(paymentIntent.ID)
	if orderID == "" {
		log.Printf("No order found for payment intent: %s", paymentIntent.ID)
		return
	}

	// Update payment status
	if err := h.paymentStore.UpdatePaymentStatus(orderID, models.PaymentStatusSucceeded); err != nil {
		log.Printf("Failed to update payment status for order %s: %v", orderID, err)
		return
	}

	// Update order status to paid
	if err := h.paymentStore.UpdateOrderStatus(orderID, models.OrderStatusPaid); err != nil {
		log.Printf("Failed to update order status for order %s: %v", orderID, err)
		return
	}

	// Log payment event
	h.paymentStore.AddPaymentEvent(models.PaymentEvent{
		OrderID:   orderID,
		EventType: "payment_succeeded",
		Status:    models.PaymentStatusSucceeded,
		Data: map[string]interface{}{
			"payment_intent_id": paymentIntent.ID,
			"amount":            paymentIntent.Amount,
			"currency":          paymentIntent.Currency,
			"payment_method":    getPaymentMethod(paymentIntent.PaymentMethod),
		},
	})

	// TODO: Trigger order fulfillment (send download links, etc.)
	log.Printf("Order %s is ready for fulfillment", orderID)
}

// handlePaymentIntentFailed processes failed payment intents
func (h *Handlers) handlePaymentIntentFailed(event stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	err := json.Unmarshal(event.Data.Raw, &paymentIntent)
	if err != nil {
		log.Printf("Error parsing payment_intent.payment_failed: %v", err)
		return
	}

	log.Printf("Payment failed: %s", paymentIntent.ID)

	orderID := h.findOrderByPaymentIntentID(paymentIntent.ID)
	if orderID == "" {
		log.Printf("No order found for payment intent: %s", paymentIntent.ID)
		return
	}

	// Update payment status
	if err := h.paymentStore.UpdatePaymentStatus(orderID, models.PaymentStatusFailed); err != nil {
		log.Printf("Failed to update payment status for order %s: %v", orderID, err)
		return
	}

	// Log payment event
	h.paymentStore.AddPaymentEvent(models.PaymentEvent{
		OrderID:   orderID,
		EventType: "payment_failed",
		Status:    models.PaymentStatusFailed,
		Data: map[string]interface{}{
			"payment_intent_id": paymentIntent.ID,
			"failure_code":      paymentIntent.LastPaymentError.Code,
			"failure_message":   paymentIntent.LastPaymentError.Message,
		},
	})
}

// handlePaymentIntentCanceled processes canceled payment intents
func (h *Handlers) handlePaymentIntentCanceled(event stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	err := json.Unmarshal(event.Data.Raw, &paymentIntent)
	if err != nil {
		log.Printf("Error parsing payment_intent.canceled: %v", err)
		return
	}

	log.Printf("Payment canceled: %s", paymentIntent.ID)

	orderID := h.findOrderByPaymentIntentID(paymentIntent.ID)
	if orderID == "" {
		log.Printf("No order found for payment intent: %s", paymentIntent.ID)
		return
	}

	// Update statuses
	h.paymentStore.UpdatePaymentStatus(orderID, models.PaymentStatusCanceled)
	h.paymentStore.UpdateOrderStatus(orderID, models.OrderStatusCanceled)

	// Log payment event
	h.paymentStore.AddPaymentEvent(models.PaymentEvent{
		OrderID:   orderID,
		EventType: "payment_canceled",
		Status:    models.PaymentStatusCanceled,
		Data: map[string]interface{}{
			"payment_intent_id": paymentIntent.ID,
			"canceled_at":       time.Now(),
		},
	})
}

// handleCheckoutSessionCompleted processes completed checkout sessions
func (h *Handlers) handleCheckoutSessionCompleted(event stripe.Event) {
	var session stripe.CheckoutSession
	err := json.Unmarshal(event.Data.Raw, &session)
	if err != nil {
		log.Printf("Error parsing checkout.session.completed: %v", err)
		return
	}

	log.Printf("Checkout session completed: %s", session.ID)

	// Find order by session ID or payment intent ID
	orderID := h.findOrderBySessionID(session.ID)
	if orderID == "" && session.PaymentIntent != nil {
		orderID = h.findOrderByPaymentIntentID(session.PaymentIntent.ID)
	}

	if orderID == "" {
		log.Printf("No order found for checkout session: %s", session.ID)
		return
	}

	// Update order with session information
	order, err := h.paymentStore.GetOrder(orderID)
	if err != nil {
		log.Printf("Failed to get order %s: %v", orderID, err)
		return
	}

	// Update customer info if we have it
	if session.CustomerDetails != nil {
		order.CustomerInfo.Email = session.CustomerDetails.Email
		if session.CustomerDetails.Name != "" {
			order.CustomerInfo.Name = session.CustomerDetails.Name
		}
		if session.CustomerDetails.Phone != "" {
			order.CustomerInfo.Phone = session.CustomerDetails.Phone
		}
	}

	// Update payment info
	if session.PaymentIntent != nil {
		order.Payment.StripePaymentIntentID = session.PaymentIntent.ID
	}
	order.Payment.StripeSessionID = session.ID

	if err := h.paymentStore.UpdateOrder(order); err != nil {
		log.Printf("Failed to update order %s: %v", orderID, err)
		return
	}

	// Log checkout event
	h.paymentStore.AddPaymentEvent(models.PaymentEvent{
		OrderID:   orderID,
		EventType: "checkout_completed",
		Status:    models.PaymentStatusSucceeded,
		Data: map[string]interface{}{
			"session_id":        session.ID,
			"payment_intent_id": session.PaymentIntent.ID,
			"customer_email":    session.CustomerDetails.Email,
		},
	})
}

// handleInvoicePaymentSucceeded processes successful invoice payments
func (h *Handlers) handleInvoicePaymentSucceeded(event stripe.Event) {
	var invoice stripe.Invoice
	err := json.Unmarshal(event.Data.Raw, &invoice)
	if err != nil {
		log.Printf("Error parsing invoice.payment_succeeded: %v", err)
		return
	}

	log.Printf("Invoice payment succeeded: %s", invoice.ID)

	// Log the event for tracking purposes
	// You might want to implement subscription or recurring payment logic here
}

// handleChargeDisputeCreated processes charge disputes
func (h *Handlers) handleChargeDisputeCreated(event stripe.Event) {
	var dispute stripe.Dispute
	err := json.Unmarshal(event.Data.Raw, &dispute)
	if err != nil {
		log.Printf("Error parsing charge.dispute.created: %v", err)
		return
	}

	log.Printf("Charge dispute created: %s for charge: %s", dispute.ID, dispute.Charge.ID)

	// Find order by charge ID or payment intent ID
	// You might need to implement additional tracking for this
	// For now, just log it for manual review

	// TODO: Implement dispute handling logic
	// - Find related order
	// - Update order status
	// - Send notification to admin
	// - Prepare dispute response materials
}

// Helper functions

// findOrderByPaymentIntentID finds an order by Stripe payment intent ID
func (h *Handlers) findOrderByPaymentIntentID(paymentIntentID string) string {
	// This is a simple implementation - in a real database, you'd do a query
	// For now, we'll iterate through orders (this should be optimized with proper indexing)

	// Get all orders and search (this is inefficient but works for the demo)
	orders, err := h.paymentStore.GetAllOrders(1000, 0) // Get a large batch
	if err != nil {
		return ""
	}

	for _, summary := range orders {
		order, err := h.paymentStore.GetOrder(summary.ID)
		if err != nil {
			continue
		}
		if order.Payment.StripePaymentIntentID == paymentIntentID {
			return order.ID
		}
	}

	return ""
}

// findOrderBySessionID finds an order by Stripe checkout session ID
func (h *Handlers) findOrderBySessionID(sessionID string) string {
	// Similar to findOrderByPaymentIntentID but searches by session ID
	orders, err := h.paymentStore.GetAllOrders(1000, 0)
	if err != nil {
		return ""
	}

	for _, summary := range orders {
		order, err := h.paymentStore.GetOrder(summary.ID)
		if err != nil {
			continue
		}
		if order.Payment.StripeSessionID == sessionID {
			return order.ID
		}
	}

	return ""
}

// getPaymentMethod extracts payment method information from Stripe payment method
func getPaymentMethod(pm *stripe.PaymentMethod) models.PaymentMethod {
	if pm == nil {
		return models.PaymentMethodCard // default
	}

	switch pm.Type {
	case stripe.PaymentMethodTypeCard:
		return models.PaymentMethodCard
	case stripe.PaymentMethodTypePaypal:
		return models.PaymentMethodPayPal
	default:
		return models.PaymentMethodCard
	}
}
