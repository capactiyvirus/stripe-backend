// models/payment.go
package models

import (
	"time"
)

type PaymentStatus string
type OrderStatus string
type PaymentMethod string

const (
	// Payment statuses
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCanceled  PaymentStatus = "canceled"
	PaymentStatusRefunded  PaymentStatus = "refunded"

	// Order statuses
	OrderStatusCreated   OrderStatus = "created"
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusFulfilled OrderStatus = "fulfilled"
	OrderStatusCanceled  OrderStatus = "canceled"
	OrderStatusRefunded  OrderStatus = "refunded"

	// Payment methods
	PaymentMethodCard      PaymentMethod = "card"
	PaymentMethodPayPal    PaymentMethod = "paypal"
	PaymentMethodApplePay  PaymentMethod = "apple_pay"
	PaymentMethodGooglePay PaymentMethod = "google_pay"
)

// Order represents a customer order
type Order struct {
	ID           string            `json:"id"`
	TrackingID   string            `json:"tracking_id"`
	CustomerInfo CustomerInfo      `json:"customer_info"`
	Items        []OrderItem       `json:"items"`
	Payment      PaymentInfo       `json:"payment"`
	Status       OrderStatus       `json:"status"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	FulfilledAt  *time.Time        `json:"fulfilled_at,omitempty"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	FileType    string  `json:"file_type"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	DownloadURL string  `json:"download_url,omitempty"`
}

// CustomerInfo holds customer details
type CustomerInfo struct {
	Email     string `json:"email"`
	Name      string `json:"name,omitempty"`
	Phone     string `json:"phone,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// PaymentInfo holds payment-related information
type PaymentInfo struct {
	StripePaymentIntentID string        `json:"stripe_payment_intent_id,omitempty"`
	StripeSessionID       string        `json:"stripe_session_id,omitempty"`
	Amount                int64         `json:"amount"` // Amount in cents
	Currency              string        `json:"currency"`
	Status                PaymentStatus `json:"status"`
	Method                PaymentMethod `json:"method,omitempty"`
	ProcessedAt           *time.Time    `json:"processed_at,omitempty"`
	RefundedAt            *time.Time    `json:"refunded_at,omitempty"`
}

// PaymentEvent represents payment status changes
type PaymentEvent struct {
	ID        string        `json:"id"`
	OrderID   string        `json:"order_id"`
	EventType string        `json:"event_type"`
	Status    PaymentStatus `json:"status"`
	Data      interface{}   `json:"data"`
	CreatedAt time.Time     `json:"created_at"`
}

// OrderSummary provides a summary view of orders
type OrderSummary struct {
	ID            string      `json:"id"`
	TrackingID    string      `json:"tracking_id"`
	CustomerEmail string      `json:"customer_email"`
	TotalAmount   float64     `json:"total_amount"`
	Status        OrderStatus `json:"status"`
	ItemCount     int         `json:"item_count"`
	CreatedAt     time.Time   `json:"created_at"`
}

// PaymentStats provides statistics about payments
type PaymentStats struct {
	TotalOrders       int     `json:"total_orders"`
	TotalRevenue      float64 `json:"total_revenue"`
	PendingOrders     int     `json:"pending_orders"`
	CompletedOrders   int     `json:"completed_orders"`
	RefundedOrders    int     `json:"refunded_orders"`
	AverageOrderValue float64 `json:"average_order_value"`
	RevenueToday      float64 `json:"revenue_today"`
	RevenueThisMonth  float64 `json:"revenue_this_month"`
}
