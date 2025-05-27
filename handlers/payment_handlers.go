// handlers/payment_handlers.go
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/capactiyvirus/stripe-backend/config"
	"github.com/capactiyvirus/stripe-backend/models"
	"github.com/capactiyvirus/stripe-backend/store"
	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

// Enhanced Handlers struct with payment store
type Handlers struct {
	config       *config.Config
	paymentStore *store.PaymentStore
}

// NewHandlers creates a new Handlers instance with payment store
func NewHandlers(cfg *config.Config) *Handlers {
	return &Handlers{
		config:       cfg,
		paymentStore: store.NewPaymentStore(),
	}
}

// Request/Response types
type CreateOrderRequest struct {
	CustomerInfo models.CustomerInfo `json:"customer_info"`
	Items        []OrderItemRequest  `json:"items"`
	Metadata     map[string]string   `json:"metadata,omitempty"`
}

type OrderItemRequest struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	FileType    string  `json:"file_type"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

type CreateOrderResponse struct {
	Order        *models.Order `json:"order"`
	ClientSecret string        `json:"client_secret,omitempty"`
	CheckoutURL  string        `json:"checkout_url,omitempty"`
}

// generateTrackingID generates a unique tracking ID
func generateTrackingID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "TRK" + hex.EncodeToString(bytes)
}

// generateOrderID generates a unique order ID
func generateOrderID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "ORD" + hex.EncodeToString(bytes)
}

// CreateOrder creates a new order with payment tracking
func (h *Handlers) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate request
	if req.CustomerInfo.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Customer email is required")
		return
	}
	if len(req.Items) == 0 {
		respondWithError(w, http.StatusBadRequest, "At least one item is required")
		return
	}

	// Calculate total amount
	var totalAmount int64
	orderItems := make([]models.OrderItem, len(req.Items))
	for i, item := range req.Items {
		if item.Quantity <= 0 {
			item.Quantity = 1
		}
		itemTotal := int64(item.Price * 100 * float64(item.Quantity)) // Convert to cents
		totalAmount += itemTotal

		orderItems[i] = models.OrderItem{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			FileType:    item.FileType,
			Price:       item.Price,
			Quantity:    item.Quantity,
		}
	}

	// Create order
	order := &models.Order{
		ID:           generateOrderID(),
		TrackingID:   generateTrackingID(),
		CustomerInfo: req.CustomerInfo,
		Items:        orderItems,
		Payment: models.PaymentInfo{
			Amount:   totalAmount,
			Currency: "usd", // Default to USD
			Status:   models.PaymentStatusPending,
		},
		Status:   models.OrderStatusCreated,
		Metadata: req.Metadata,
	}

	// Store the order
	if err := h.paymentStore.CreateOrder(order); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create order: "+err.Error())
		return
	}

	// Create Stripe payment intent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(totalAmount),
		Currency: stripe.String("usd"),
		Metadata: map[string]string{
			"order_id":       order.ID,
			"tracking_id":    order.TrackingID,
			"customer_email": req.CustomerInfo.Email,
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create payment intent: "+err.Error())
		return
	}

	// Update order with payment intent ID
	order.Payment.StripePaymentIntentID = pi.ID
	order.Status = models.OrderStatusPending
	if err := h.paymentStore.UpdateOrder(order); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update order: "+err.Error())
		return
	}

	// Log payment event
	h.paymentStore.AddPaymentEvent(models.PaymentEvent{
		OrderID:   order.ID,
		EventType: "order_created",
		Status:    models.PaymentStatusPending,
		Data:      map[string]interface{}{"payment_intent_id": pi.ID},
	})

	response := CreateOrderResponse{
		Order:        order,
		ClientSecret: pi.ClientSecret,
	}

	respondWithJSON(w, http.StatusCreated, response)
}

// GetPaymentStatus gets the current status of a payment by order ID
func (h *Handlers) GetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	if orderID == "" {
		respondWithError(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	order, err := h.paymentStore.GetOrder(orderID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Order not found")
		return
	}

	// If we have a Stripe payment intent, sync the status
	if order.Payment.StripePaymentIntentID != "" {
		pi, err := paymentintent.Get(order.Payment.StripePaymentIntentID, nil)
		if err == nil {
			// Update our local status if it differs
			stripeStatus := convertStripeStatus(string(pi.Status))
			if stripeStatus != order.Payment.Status {
				h.paymentStore.UpdatePaymentStatus(order.ID, stripeStatus)
				order.Payment.Status = stripeStatus
			}
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"order_id":       order.ID,
		"tracking_id":    order.TrackingID,
		"payment_status": order.Payment.Status,
		"order_status":   order.Status,
		"amount":         order.Payment.Amount,
		"currency":       order.Payment.Currency,
		"created_at":     order.CreatedAt,
		"updated_at":     order.UpdatedAt,
	})
}

// GetOrderDetails retrieves full order details
func (h *Handlers) GetOrderDetails(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	if orderID == "" {
		respondWithError(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	order, err := h.paymentStore.GetOrder(orderID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Order not found")
		return
	}

	respondWithJSON(w, http.StatusOK, order)
}

// TrackPayment tracks a payment by tracking ID
func (h *Handlers) TrackPayment(w http.ResponseWriter, r *http.Request) {
	trackingID := chi.URLParam(r, "trackingID")
	if trackingID == "" {
		respondWithError(w, http.StatusBadRequest, "Tracking ID is required")
		return
	}

	order, err := h.paymentStore.GetOrderByTrackingID(trackingID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Order not found")
		return
	}

	// Get payment events
	events, _ := h.paymentStore.GetPaymentEvents(order.ID)

	response := map[string]interface{}{
		"order":  order,
		"events": events,
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetCustomerPayments retrieves all payments for a customer
func (h *Handlers) GetCustomerPayments(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")
	if email == "" {
		respondWithError(w, http.StatusBadRequest, "Customer email is required")
		return
	}

	orders, err := h.paymentStore.GetCustomerOrders(email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve customer orders")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"customer_email": email,
		"orders":         orders,
		"total_orders":   len(orders),
	})
}

// GetAllPayments retrieves all payments (admin endpoint)
func (h *Handlers) GetAllPayments(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	orders, err := h.paymentStore.GetAllOrders(limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve orders")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"orders": orders,
		"limit":  limit,
		"offset": offset,
	})
}

// GetPaymentStats retrieves payment statistics
func (h *Handlers) GetPaymentStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.paymentStore.GetPaymentStats()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve payment stats")
		return
	}

	respondWithJSON(w, http.StatusOK, stats)
}

// FulfillOrder marks an order as fulfilled
func (h *Handlers) FulfillOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	if orderID == "" {
		respondWithError(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	// Check if order exists and is paid
	order, err := h.paymentStore.GetOrder(orderID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Order not found")
		return
	}

	if order.Status != models.OrderStatusPaid {
		respondWithError(w, http.StatusBadRequest, "Order must be paid before fulfillment")
		return
	}

	// Update order status
	if err := h.paymentStore.UpdateOrderStatus(orderID, models.OrderStatusFulfilled); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fulfill order")
		return
	}

	// Log fulfillment event
	h.paymentStore.AddPaymentEvent(models.PaymentEvent{
		OrderID:   orderID,
		EventType: "order_fulfilled",
		Status:    models.PaymentStatusSucceeded,
		Data:      map[string]interface{}{"fulfilled_at": time.Now()},
	})

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message":  "Order fulfilled successfully",
		"order_id": orderID,
	})
}

// RefundOrder processes a refund for an order
func (h *Handlers) RefundOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	if orderID == "" {
		respondWithError(w, http.StatusBadRequest, "Order ID is required")
		return
	}

	order, err := h.paymentStore.GetOrder(orderID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Order not found")
		return
	}

	if order.Payment.StripePaymentIntentID == "" {
		respondWithError(w, http.StatusBadRequest, "No payment intent found for this order")
		return
	}

	// Process refund with Stripe (implement based on your needs)
	// For now, just update the status
	if err := h.paymentStore.UpdateOrderStatus(orderID, models.OrderStatusRefunded); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process refund")
		return
	}

	if err := h.paymentStore.UpdatePaymentStatus(orderID, models.PaymentStatusRefunded); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update payment status")
		return
	}

	// Log refund event
	h.paymentStore.AddPaymentEvent(models.PaymentEvent{
		OrderID:   orderID,
		EventType: "order_refunded",
		Status:    models.PaymentStatusRefunded,
		Data:      map[string]interface{}{"refunded_at": time.Now()},
	})

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message":  "Order refunded successfully",
		"order_id": orderID,
	})
}

// convertStripeStatus converts Stripe payment intent status to our internal status
func convertStripeStatus(stripeStatus string) models.PaymentStatus {
	switch stripeStatus {
	case "succeeded":
		return models.PaymentStatusSucceeded
	case "canceled":
		return models.PaymentStatusCanceled
	case "processing", "requires_payment_method", "requires_confirmation", "requires_action":
		return models.PaymentStatusPending
	default:
		return models.PaymentStatusFailed
	}
}

// Helper functions (keep existing ones and add these)
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error encoding response"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
