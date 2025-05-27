// store/postgres_store.go
package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/capactiyvirus/stripe-backend/models"
	_ "github.com/lib/pq"
)

// PostgresStore implements PaymentStore interface using PostgreSQL
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresStore{db: db}, nil
}

// Close closes the database connection
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// CreateOrder creates a new order in the database
func (s *PostgresStore) CreateOrder(order *models.Order) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert order
	orderQuery := `
		INSERT INTO orders (
			id, tracking_id, customer_email, customer_name, customer_phone, 
			customer_ip_address, status, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = tx.Exec(orderQuery,
		order.ID,
		order.TrackingID,
		order.CustomerInfo.Email,
		order.CustomerInfo.Name,
		order.CustomerInfo.Phone,
		order.CustomerInfo.IPAddress,
		order.Status,
		order.Metadata,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// Insert order items
	itemQuery := `
		INSERT INTO order_items (
			order_id, product_id, product_name, file_type, price, quantity
		) VALUES ($1, $2, $3, $4, $5, $6)`

	for _, item := range order.Items {
		_, err = tx.Exec(itemQuery,
			order.ID,
			item.ProductID,
			item.ProductName,
			item.FileType,
			item.Price,
			item.Quantity,
		)
		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	// Insert payment record
	paymentQuery := `
		INSERT INTO payments (
			order_id, stripe_payment_intent_id, stripe_session_id, 
			amount, currency, status, method
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = tx.Exec(paymentQuery,
		order.ID,
		order.Payment.StripePaymentIntentID,
		order.Payment.StripeSessionID,
		order.Payment.Amount,
		order.Payment.Currency,
		order.Payment.Status,
		order.Payment.Method,
	)
	if err != nil {
		return fmt.Errorf("failed to insert payment: %w", err)
	}

	return tx.Commit()
}

// GetOrder retrieves an order by ID
func (s *PostgresStore) GetOrder(orderID string) (*models.Order, error) {
	orderQuery := `
		SELECT 
			o.id, o.tracking_id, o.customer_email, o.customer_name, o.customer_phone,
			o.customer_ip_address, o.status, o.metadata, o.created_at, o.updated_at, o.fulfilled_at,
			p.stripe_payment_intent_id, p.stripe_session_id, p.amount, p.currency, 
			p.status, p.method, p.processed_at, p.refunded_at
		FROM orders o
		LEFT JOIN payments p ON o.id = p.order_id
		WHERE o.id = $1`

	var order models.Order
	var fulfilledAt, processedAt, refundedAt *time.Time

	err := s.db.QueryRow(orderQuery, orderID).Scan(
		&order.ID,
		&order.TrackingID,
		&order.CustomerInfo.Email,
		&order.CustomerInfo.Name,
		&order.CustomerInfo.Phone,
		&order.CustomerInfo.IPAddress,
		&order.Status,
		&order.Metadata,
		&order.CreatedAt,
		&order.UpdatedAt,
		&fulfilledAt,
		&order.Payment.StripePaymentIntentID,
		&order.Payment.StripeSessionID,
		&order.Payment.Amount,
		&order.Payment.Currency,
		&order.Payment.Status,
		&order.Payment.Method,
		&processedAt,
		&refundedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found: %s", orderID)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Set nullable time fields
	order.FulfilledAt = fulfilledAt
	order.Payment.ProcessedAt = processedAt
	order.Payment.RefundedAt = refundedAt

	// Get order items
	itemsQuery := `
		SELECT product_id, product_name, file_type, price, quantity, download_url
		FROM order_items 
		WHERE order_id = $1`

	rows, err := s.db.Query(itemsQuery, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(
			&item.ProductID,
			&item.ProductName,
			&item.FileType,
			&item.Price,
			&item.Quantity,
			&item.DownloadURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}

	order.Items = items
	return &order, nil
}

// GetOrderByTrackingID retrieves an order by tracking ID
func (s *PostgresStore) GetOrderByTrackingID(trackingID string) (*models.Order, error) {
	orderQuery := `
		SELECT id FROM orders WHERE tracking_id = $1`

	var orderID string
	err := s.db.QueryRow(orderQuery, trackingID).Scan(&orderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found with tracking ID: %s", trackingID)
		}
		return nil, fmt.Errorf("failed to get order by tracking ID: %w", err)
	}

	return s.GetOrder(orderID)
}

// UpdateOrder updates an existing order
func (s *PostgresStore) UpdateOrder(order *models.Order) error {
	orderQuery := `
		UPDATE orders SET 
			customer_email = $2, customer_name = $3, customer_phone = $4,
			customer_ip_address = $5, status = $6, metadata = $7, updated_at = $8,
			fulfilled_at = $9
		WHERE id = $1`

	_, err := s.db.Exec(orderQuery,
		order.ID,
		order.CustomerInfo.Email,
		order.CustomerInfo.Name,
		order.CustomerInfo.Phone,
		order.CustomerInfo.IPAddress,
		order.Status,
		order.Metadata,
		time.Now(),
		order.FulfilledAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	// Update payment
	paymentQuery := `
		UPDATE payments SET 
			stripe_payment_intent_id = $2, stripe_session_id = $3,
			amount = $4, currency = $5, status = $6, method = $7,
			processed_at = $8, refunded_at = $9
		WHERE order_id = $1`

	_, err = s.db.Exec(paymentQuery,
		order.ID,
		order.Payment.StripePaymentIntentID,
		order.Payment.StripeSessionID,
		order.Payment.Amount,
		order.Payment.Currency,
		order.Payment.Status,
		order.Payment.Method,
		order.Payment.ProcessedAt,
		order.Payment.RefundedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	return nil
}

// UpdateOrderStatus updates the status of an order
func (s *PostgresStore) UpdateOrderStatus(orderID string, status models.OrderStatus) error {
	query := `UPDATE orders SET status = $2, updated_at = $3`
	args := []interface{}{orderID, status, time.Now()}

	// Set fulfilled_at if status is fulfilled
	if status == models.OrderStatusFulfilled {
		query += `, fulfilled_at = $4`
		args = append(args, time.Now())
	}

	query += ` WHERE id = $1`

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// UpdatePaymentStatus updates the payment status of an order
func (s *PostgresStore) UpdatePaymentStatus(orderID string, status models.PaymentStatus) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update payment status
	paymentQuery := `UPDATE payments SET status = $2`
	args := []interface{}{orderID, status}

	if status == models.PaymentStatusSucceeded {
		paymentQuery += `, processed_at = $3`
		args = append(args, time.Now())
	} else if status == models.PaymentStatusRefunded {
		paymentQuery += `, refunded_at = $3`
		args = append(args, time.Now())
	}

	paymentQuery += ` WHERE order_id = $1`

	_, err = tx.Exec(paymentQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// Update order status if payment succeeded
	if status == models.PaymentStatusSucceeded {
		orderQuery := `UPDATE orders SET status = $2, updated_at = $3 WHERE id = $1`
		_, err = tx.Exec(orderQuery, orderID, models.OrderStatusPaid, time.Now())
		if err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}
	}

	return tx.Commit()
}

// GetCustomerOrders retrieves all orders for a customer by email
func (s *PostgresStore) GetCustomerOrders(email string) ([]*models.Order, error) {
	query := `
		SELECT id FROM orders 
		WHERE customer_email = $1 
		ORDER BY created_at DESC`

	rows, err := s.db.Query(query, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var orderID string
		if err := rows.Scan(&orderID); err != nil {
			return nil, fmt.Errorf("failed to scan order ID: %w", err)
		}

		order, err := s.GetOrder(orderID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order %s: %w", orderID, err)
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// GetAllOrders retrieves all orders with pagination
func (s *PostgresStore) GetAllOrders(limit, offset int) ([]*models.OrderSummary, error) {
	query := `
		SELECT 
			o.id, o.tracking_id, o.customer_email, o.status, o.created_at,
			p.amount, COUNT(oi.id) as item_count
		FROM orders o
		LEFT JOIN payments p ON o.id = p.order_id
		LEFT JOIN order_items oi ON o.id = oi.order_id
		GROUP BY o.id, o.tracking_id, o.customer_email, o.status, o.created_at, p.amount
		ORDER BY o.created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get all orders: %w", err)
	}
	defer rows.Close()

	var summaries []*models.OrderSummary
	for rows.Next() {
		var summary models.OrderSummary
		var amount int64
		var itemCount int

		err := rows.Scan(
			&summary.ID,
			&summary.TrackingID,
			&summary.CustomerEmail,
			&summary.Status,
			&summary.CreatedAt,
			&amount,
			&itemCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order summary: %w", err)
		}

		summary.TotalAmount = float64(amount) / 100
		summary.ItemCount = itemCount
		summaries = append(summaries, &summary)
	}

	return summaries, nil
}

// AddPaymentEvent adds a payment event
func (s *PostgresStore) AddPaymentEvent(event models.PaymentEvent) error {
	query := `
		INSERT INTO payment_events (order_id, event_type, status, data)
		VALUES ($1, $2, $3, $4)`

	_, err := s.db.Exec(query, event.OrderID, event.EventType, event.Status, event.Data)
	if err != nil {
		return fmt.Errorf("failed to add payment event: %w", err)
	}

	return nil
}

// GetPaymentEvents retrieves payment events for an order
func (s *PostgresStore) GetPaymentEvents(orderID string) ([]models.PaymentEvent, error) {
	query := `
		SELECT id, order_id, event_type, status, data, created_at
		FROM payment_events 
		WHERE order_id = $1 
		ORDER BY created_at ASC`

	rows, err := s.db.Query(query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment events: %w", err)
	}
	defer rows.Close()

	var events []models.PaymentEvent
	for rows.Next() {
		var event models.PaymentEvent
		err := rows.Scan(
			&event.ID,
			&event.OrderID,
			&event.EventType,
			&event.Status,
			&event.Data,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// GetPaymentStats calculates payment statistics
func (s *PostgresStore) GetPaymentStats() (*models.PaymentStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_orders,
			COALESCE(SUM(CASE WHEN o.status IN ('paid', 'fulfilled') THEN p.amount ELSE 0 END), 0) as total_revenue,
			COUNT(CASE WHEN o.status = 'pending' THEN 1 END) as pending_orders,
			COUNT(CASE WHEN o.status IN ('paid', 'fulfilled') THEN 1 END) as completed_orders,
			COUNT(CASE WHEN o.status = 'refunded' THEN 1 END) as refunded_orders,
			COALESCE(SUM(CASE WHEN DATE(o.created_at) = CURRENT_DATE AND o.status IN ('paid', 'fulfilled') THEN p.amount ELSE 0 END), 0) as revenue_today,
			COALESCE(SUM(CASE WHEN DATE_TRUNC('month', o.created_at) = DATE_TRUNC('month', CURRENT_DATE) AND o.status IN ('paid', 'fulfilled') THEN p.amount ELSE 0 END), 0) as revenue_this_month
		FROM orders o
		LEFT JOIN payments p ON o.id = p.order_id`

	var stats models.PaymentStats
	var totalRevenue, revenueToday, revenueThisMonth int64

	err := s.db.QueryRow(query).Scan(
		&stats.TotalOrders,
		&totalRevenue,
		&stats.PendingOrders,
		&stats.CompletedOrders,
		&stats.RefundedOrders,
		&revenueToday,
		&revenueThisMonth,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment stats: %w", err)
	}

	// Convert from cents to dollars
	stats.TotalRevenue = float64(totalRevenue) / 100
	stats.RevenueToday = float64(revenueToday) / 100
	stats.RevenueThisMonth = float64(revenueThisMonth) / 100

	// Calculate average order value
	if stats.CompletedOrders > 0 {
		stats.AverageOrderValue = stats.TotalRevenue / float64(stats.CompletedOrders)
	}

	return &stats, nil
}

// FindOrderByPaymentIntentID finds an order by Stripe payment intent ID
func (s *PostgresStore) FindOrderByPaymentIntentID(paymentIntentID string) (string, error) {
	query := `SELECT order_id FROM payments WHERE stripe_payment_intent_id = $1`

	var orderID string
	err := s.db.QueryRow(query, paymentIntentID).Scan(&orderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("order not found for payment intent: %s", paymentIntentID)
		}
		return "", fmt.Errorf("failed to find order by payment intent ID: %w", err)
	}

	return orderID, nil
}

// FindOrderBySessionID finds an order by Stripe checkout session ID
func (s *PostgresStore) FindOrderBySessionID(sessionID string) (string, error) {
	query := `SELECT order_id FROM payments WHERE stripe_session_id = $1`

	var orderID string
	err := s.db.QueryRow(query, sessionID).Scan(&orderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("order not found for session: %s", sessionID)
		}
		return "", fmt.Errorf("failed to find order by session ID: %w", err)
	}

	return orderID, nil
}
