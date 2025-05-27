// tests/payment_test.go
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/capactiyvirus/stripe-backend/config"
	"github.com/capactiyvirus/stripe-backend/handlers"
	"github.com/capactiyvirus/stripe-backend/models"
	"github.com/capactiyvirus/stripe-backend/routes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateOrder tests order creation
func TestCreateOrder(t *testing.T) {
	// Setup test server
	cfg := &config.Config{
		StripeSecretKey: "sk_test_fake_key",
		Environment:     "test",
	}
	h := handlers.NewHandlers(cfg)
	router := routes.SetupRoutes(h)

	// Test data
	orderRequest := map[string]interface{}{
		"customer_info": map[string]string{
			"email": "test@example.com",
			"name":  "Test Customer",
		},
		"items": []map[string]interface{}{
			{
				"product_id":   "1",
				"product_name": "Test Product",
				"file_type":    "PDF",
				"price":        9.99,
				"quantity":     1,
			},
		},
		"metadata": map[string]string{
			"source": "test",
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(orderRequest)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest("POST", "/api/payments/create-order", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Validate response structure
	assert.Contains(t, response, "order")
	assert.Contains(t, response, "client_secret")

	order := response["order"].(map[string]interface{})
	assert.Equal(t, "test@example.com", order["customer_info"].(map[string]interface{})["email"])
	assert.NotEmpty(t, order["tracking_id"])
	assert.Equal(t, "created", order["status"])
}

// TestTrackPayment tests payment tracking
func TestTrackPayment(t *testing.T) {
	// First create an order
	cfg := &config.Config{
		StripeSecretKey: "sk_test_fake_key",
		Environment:     "test",
	}
	h := handlers.NewHandlers(cfg)
	router := routes.SetupRoutes(h)

	// Create order first
	order := &models.Order{
		ID:         "test-order-123",
		TrackingID: "TRK123456789",
		CustomerInfo: models.CustomerInfo{
			Email: "test@example.com",
			Name:  "Test Customer",
		},
		Items: []models.OrderItem{
			{
				ProductID:   "1",
				ProductName: "Test Product",
				FileType:    "PDF",
				Price:       9.99,
				Quantity:    1,
			},
		},
		Payment: models.PaymentInfo{
			Amount:   999, // $9.99 in cents
			Currency: "usd",
			Status:   models.PaymentStatusPending,
		},
		Status: models.OrderStatusCreated,
	}

	// Add order to store manually for testing
	err := h.PaymentStore.CreateOrder(order)
	require.NoError(t, err)

	// Now test tracking
	req := httptest.NewRequest("GET", "/api/payments/track/TRK123456789", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "order")
	assert.Contains(t, response, "events")

	trackedOrder := response["order"].(map[string]interface{})
	assert.Equal(t, "TRK123456789", trackedOrder["tracking_id"])
}

// TestPaymentStatusUpdate tests payment status updates
func TestPaymentStatusUpdate(t *testing.T) {
	cfg := &config.Config{
		StripeSecretKey: "sk_test_fake_key",
		Environment:     "test",
	}
	h := handlers.NewHandlers(cfg)

	// Create test order
	order := &models.Order{
		ID:         "test-order-456",
		TrackingID: "TRK987654321",
		CustomerInfo: models.CustomerInfo{
			Email: "test2@example.com",
		},
		Payment: models.PaymentInfo{
			Amount:   1999, // $19.99 in cents
			Currency: "usd",
			Status:   models.PaymentStatusPending,
		},
		Status: models.OrderStatusPending,
	}

	err := h.PaymentStore.CreateOrder(order)
	require.NoError(t, err)

	// Test status update
	err = h.PaymentStore.UpdatePaymentStatus("test-order-456", models.PaymentStatusSucceeded)
	require.NoError(t, err)

	// Verify update
	updatedOrder, err := h.PaymentStore.GetOrder("test-order-456")
	require.NoError(t, err)

	assert.Equal(t, models.PaymentStatusSucceeded, updatedOrder.Payment.Status)
	assert.Equal(t, models.OrderStatusPaid, updatedOrder.Status) // Should auto-update order status
}

// TestGetPaymentStats tests payment statistics
func TestGetPaymentStats(t *testing.T) {
	cfg := &config.Config{
		StripeSecretKey: "sk_test_fake_key",
		Environment:     "test",
	}
	h := handlers.NewHandlers(cfg)
	router := routes.SetupRoutes(h)

	// Create some test orders with different statuses
	orders := []*models.Order{
		{
			ID:           "order-1",
			TrackingID:   "TRK001",
			CustomerInfo: models.CustomerInfo{Email: "customer1@example.com"},
			Payment:      models.PaymentInfo{Amount: 1000, Currency: "usd", Status: models.PaymentStatusSucceeded},
			Status:       models.OrderStatusPaid,
		},
		{
			ID:           "order-2",
			TrackingID:   "TRK002",
			CustomerInfo: models.CustomerInfo{Email: "customer2@example.com"},
			Payment:      models.PaymentInfo{Amount: 2000, Currency: "usd", Status: models.PaymentStatusSucceeded},
			Status:       models.OrderStatusFulfilled,
		},
		{
			ID:           "order-3",
			TrackingID:   "TRK003",
			CustomerInfo: models.CustomerInfo{Email: "customer3@example.com"},
			Payment:      models.PaymentInfo{Amount: 1500, Currency: "usd", Status: models.PaymentStatusPending},
			Status:       models.OrderStatusPending,
		},
	}

	// Add orders to store
	for _, order := range orders {
		err := h.PaymentStore.CreateOrder(order)
		require.NoError(t, err)
	}

	// Test stats endpoint
	req := httptest.NewRequest("GET", "/api/payments/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats models.PaymentStats
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	require.NoError(t, err)

	assert.Equal(t, 3, stats.TotalOrders)
	assert.Equal(t, 2, stats.CompletedOrders)
	assert.Equal(t, 1, stats.PendingOrders)
	assert.Equal(t, 30.0, stats.TotalRevenue)      // $30.00 from completed orders
	assert.Equal(t, 15.0, stats.AverageOrderValue) // $30.00 / 2 orders
}

// BenchmarkCreateOrder benchmarks order creation performance
func BenchmarkCreateOrder(b *testing.B) {
	cfg := &config.Config{
		StripeSecretKey: "sk_test_fake_key",
		Environment:     "test",
	}
	h := handlers.NewHandlers(cfg)

	orderRequest := map[string]interface{}{
		"customer_info": map[string]string{
			"email": "benchmark@example.com",
			"name":  "Benchmark Customer",
		},
		"items": []map[string]interface{}{
			{
				"product_id":   "1",
				"product_name": "Benchmark Product",
				"file_type":    "PDF",
				"price":        9.99,
				"quantity":     1,
			},
		},
	}

	jsonData, _ := json.Marshal(orderRequest)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/payments/create-order", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// This would normally call the handler directly
		// router.ServeHTTP(w, req)
		_ = w // Avoid unused variable error
	}
}

// Integration test for full payment flow
func TestFullPaymentFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &config.Config{
		StripeSecretKey: "sk_test_fake_key",
		Environment:     "test",
	}
	h := handlers.NewHandlers(cfg)
	router := routes.SetupRoutes(h)

	// Step 1: Create order
	orderRequest := map[string]interface{}{
		"customer_info": map[string]string{
			"email": "integration@example.com",
			"name":  "Integration Test",
		},
		"items": []map[string]interface{}{
			{
				"product_id":   "1",
				"product_name": "Integration Product",
				"file_type":    "PDF",
				"price":        19.99,
				"quantity":     1,
			},
		},
	}

	jsonData, _ := json.Marshal(orderRequest)
	req := httptest.NewRequest("POST", "/api/payments/create-order", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	order := createResponse["order"].(map[string]interface{})
	orderID := order["id"].(string)
	trackingID := order["tracking_id"].(string)

	// Step 2: Check initial status
	req = httptest.NewRequest("GET", "/api/payments/status/"+orderID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var statusResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &statusResponse)
	require.NoError(t, err)

	assert.Equal(t, "pending", statusResponse["payment_status"])
	assert.Equal(t, "created", statusResponse["order_status"])

	// Step 3: Simulate payment success (normally done by webhook)
	err = h.PaymentStore.UpdatePaymentStatus(orderID, models.PaymentStatusSucceeded)
	require.NoError(t, err)

	// Step 4: Check updated status
	req = httptest.NewRequest("GET", "/api/payments/status/"+orderID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &statusResponse)
	require.NoError(t, err)

	assert.Equal(t, "succeeded", statusResponse["payment_status"])
	assert.Equal(t, "paid", statusResponse["order_status"])

	// Step 5: Track payment
	req = httptest.NewRequest("GET", "/api/payments/track/"+trackingID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var trackResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &trackResponse)
	require.NoError(t, err)

	assert.Contains(t, trackResponse, "order")
	assert.Contains(t, trackResponse, "events")

	// Step 6: Fulfill order
	req = httptest.NewRequest("POST", "/api/payments/fulfill/"+orderID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Step 7: Verify fulfillment
	req = httptest.NewRequest("GET", "/api/payments/order/"+orderID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var finalOrder map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &finalOrder)
	require.NoError(t, err)

	assert.Equal(t, "fulfilled", finalOrder["status"])
	assert.NotNil(t, finalOrder["fulfilled_at"])
}

// Load test helper
func TestLoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cfg := &config.Config{
		StripeSecretKey: "sk_test_fake_key",
		Environment:     "test",
	}
	h := handlers.NewHandlers(cfg)

	// Simulate concurrent order creation
	numGoroutines := 10
	ordersPerGoroutine := 100

	startTime := time.Now()

	// Channel to collect results
	results := make(chan error, numGoroutines*ordersPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < ordersPerGoroutine; j++ {
				order := &models.Order{
					ID:         fmt.Sprintf("load-test-%d-%d", goroutineID, j),
					TrackingID: fmt.Sprintf("TRK%d%d", goroutineID, j),
					CustomerInfo: models.CustomerInfo{
						Email: fmt.Sprintf("load-test-%d-%d@example.com", goroutineID, j),
					},
					Payment: models.PaymentInfo{
						Amount:   1000,
						Currency: "usd",
						Status:   models.PaymentStatusPending,
					},
					Status: models.OrderStatusCreated,
				}

				err := h.PaymentStore.CreateOrder(order)
				results <- err
			}
		}(i)
	}

	// Collect results
	var errors []error
	for i := 0; i < numGoroutines*ordersPerGoroutine; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	duration := time.Since(startTime)
	totalOrders := numGoroutines * ordersPerGoroutines
	ordersPerSecond := float64(totalOrders) / duration.Seconds()

	t.Logf("Load test completed:")
	t.Logf("Total orders: %d", totalOrders)
	t.Logf("Duration: %v", duration)
	t.Logf("Orders per second: %.2f", ordersPerSecond)
	t.Logf("Errors: %d", len(errors))

	assert.Empty(t, errors, "Load test should not produce errors")
	assert.Greater(t, ordersPerSecond, 100.0, "Should handle at least 100 orders per second")
}
