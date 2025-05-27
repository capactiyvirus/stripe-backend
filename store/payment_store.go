// store/payment_store.go
package store

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/capactiyvirus/stripe-backend/models"
)

// PaymentStore handles storage operations for payments and orders
type PaymentStore struct {
	orders        map[string]*models.Order
	events        map[string][]models.PaymentEvent
	trackingIDs   map[string]string   // trackingID -> orderID
	customerIndex map[string][]string // email -> []orderID
	mu            sync.RWMutex
}

// NewPaymentStore creates a new payment store
func NewPaymentStore() *PaymentStore {
	return &PaymentStore{
		orders:        make(map[string]*models.Order),
		events:        make(map[string][]models.PaymentEvent),
		trackingIDs:   make(map[string]string),
		customerIndex: make(map[string][]string),
	}
}

// CreateOrder creates a new order
func (s *PaymentStore) CreateOrder(order *models.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if order.ID == "" {
		return fmt.Errorf("order ID cannot be empty")
	}

	// Set timestamps
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now

	// Store the order
	s.orders[order.ID] = order

	// Index by tracking ID
	if order.TrackingID != "" {
		s.trackingIDs[order.TrackingID] = order.ID
	}

	// Index by customer email
	if order.CustomerInfo.Email != "" {
		s.customerIndex[order.CustomerInfo.Email] = append(
			s.customerIndex[order.CustomerInfo.Email],
			order.ID,
		)
	}

	return nil
}

// GetOrder retrieves an order by ID
func (s *PaymentStore) GetOrder(orderID string) (*models.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, exists := s.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	// Return a copy to prevent external modifications
	orderCopy := *order
	return &orderCopy, nil
}

// GetOrderByTrackingID retrieves an order by tracking ID
func (s *PaymentStore) GetOrderByTrackingID(trackingID string) (*models.Order, error) {
	s.mu.RLock()
	orderID, exists := s.trackingIDs[trackingID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("order not found with tracking ID: %s", trackingID)
	}

	return s.GetOrder(orderID)
}

// UpdateOrder updates an existing order
func (s *PaymentStore) UpdateOrder(order *models.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.orders[order.ID]; !exists {
		return fmt.Errorf("order not found: %s", order.ID)
	}

	order.UpdatedAt = time.Now()
	s.orders[order.ID] = order

	return nil
}

// UpdateOrderStatus updates the status of an order
func (s *PaymentStore) UpdateOrderStatus(orderID string, status models.OrderStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.Status = status
	order.UpdatedAt = time.Now()

	// Set fulfilled timestamp if order is fulfilled
	if status == models.OrderStatusFulfilled && order.FulfilledAt == nil {
		now := time.Now()
		order.FulfilledAt = &now
	}

	return nil
}

// UpdatePaymentStatus updates the payment status of an order
func (s *PaymentStore) UpdatePaymentStatus(orderID string, status models.PaymentStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[orderID]
	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.Payment.Status = status
	order.UpdatedAt = time.Now()

	// Update processed timestamp
	if status == models.PaymentStatusSucceeded && order.Payment.ProcessedAt == nil {
		now := time.Now()
		order.Payment.ProcessedAt = &now
		// Also update order status to paid
		order.Status = models.OrderStatusPaid
	}

	return nil
}

// GetCustomerOrders retrieves all orders for a customer by email
func (s *PaymentStore) GetCustomerOrders(email string) ([]*models.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orderIDs, exists := s.customerIndex[email]
	if !exists {
		return []*models.Order{}, nil
	}

	orders := make([]*models.Order, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		if order, exists := s.orders[orderID]; exists {
			orderCopy := *order
			orders = append(orders, &orderCopy)
		}
	}

	// Sort by creation date (newest first)
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].CreatedAt.After(orders[j].CreatedAt)
	})

	return orders, nil
}

// GetAllOrders retrieves all orders with optional pagination
func (s *PaymentStore) GetAllOrders(limit, offset int) ([]*models.OrderSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert to slice for sorting
	orderList := make([]*models.Order, 0, len(s.orders))
	for _, order := range s.orders {
		orderList = append(orderList, order)
	}

	// Sort by creation date (newest first)
	sort.Slice(orderList, func(i, j int) bool {
		return orderList[i].CreatedAt.After(orderList[j].CreatedAt)
	})

	// Apply pagination
	start := offset
	if start > len(orderList) {
		start = len(orderList)
	}

	end := start + limit
	if end > len(orderList) {
		end = len(orderList)
	}

	// Convert to summaries
	summaries := make([]*models.OrderSummary, 0, end-start)
	for i := start; i < end; i++ {
		order := orderList[i]
		totalAmount := float64(order.Payment.Amount) / 100 // Convert from cents

		summary := &models.OrderSummary{
			ID:            order.ID,
			TrackingID:    order.TrackingID,
			CustomerEmail: order.CustomerInfo.Email,
			TotalAmount:   totalAmount,
			Status:        order.Status,
			ItemCount:     len(order.Items),
			CreatedAt:     order.CreatedAt,
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// AddPaymentEvent adds a payment event
func (s *PaymentStore) AddPaymentEvent(event models.PaymentEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event.ID == "" {
		event.ID = fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	event.CreatedAt = time.Now()

	s.events[event.OrderID] = append(s.events[event.OrderID], event)
	return nil
}

// GetPaymentEvents retrieves payment events for an order
func (s *PaymentStore) GetPaymentEvents(orderID string) ([]models.PaymentEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, exists := s.events[orderID]
	if !exists {
		return []models.PaymentEvent{}, nil
	}

	// Return a copy
	eventsCopy := make([]models.PaymentEvent, len(events))
	copy(eventsCopy, events)

	return eventsCopy, nil
}

// GetPaymentStats calculates payment statistics
func (s *PaymentStore) GetPaymentStats() (*models.PaymentStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &models.PaymentStats{}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var totalRevenue float64
	var revenueToday float64
	var revenueThisMonth float64

	for _, order := range s.orders {
		stats.TotalOrders++

		orderAmount := float64(order.Payment.Amount) / 100

		switch order.Status {
		case models.OrderStatusPending:
			stats.PendingOrders++
		case models.OrderStatusPaid, models.OrderStatusFulfilled:
			stats.CompletedOrders++
			totalRevenue += orderAmount

			if order.CreatedAt.After(today) {
				revenueToday += orderAmount
			}
			if order.CreatedAt.After(thisMonth) {
				revenueThisMonth += orderAmount
			}
		case models.OrderStatusRefunded:
			stats.RefundedOrders++
		}
	}

	stats.TotalRevenue = totalRevenue
	stats.RevenueToday = revenueToday
	stats.RevenueThisMonth = revenueThisMonth

	if stats.CompletedOrders > 0 {
		stats.AverageOrderValue = totalRevenue / float64(stats.CompletedOrders)
	}

	return stats, nil
}
