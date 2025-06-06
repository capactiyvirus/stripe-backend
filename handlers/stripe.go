// handlers/stripe.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/product"
)

// Response types
type ErrorResponse struct {
	Error string `json:"error"`
}

type PaymentIntentResponse struct {
	ClientSecret string `json:"clientSecret"`
	ID           string `json:"id"`
}

type CheckoutResponse struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

// CreatePaymentIntent creates a Stripe payment intent
func (h *Handlers) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Amount      int64             `json:"amount"`
		Currency    string            `json:"currency"`
		Description string            `json:"description"`
		Metadata    map[string]string `json:"metadata"`
	}

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Default values
	if data.Currency == "" {
		data.Currency = "usd"
	}

	// Create payment intent
	params := &stripe.PaymentIntentParams{
		Amount:      stripe.Int64(data.Amount),
		Currency:    stripe.String(data.Currency),
		Description: stripe.String(data.Description),
	}

	// Add metadata if provided
	if data.Metadata != nil {
		params.Metadata = make(map[string]string)
		for k, v := range data.Metadata {
			params.Metadata[k] = v
		}
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return client secret
	respondWithJSON(w, http.StatusOK, PaymentIntentResponse{
		ClientSecret: pi.ClientSecret,
		ID:           pi.ID,
	})
}

// CreateCheckoutSession creates a Stripe checkout session
func (h *Handlers) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var data struct {
		ProductName string `json:"productName"`
		Amount      int64  `json:"amount"`
		Currency    string `json:"currency"`
		SuccessURL  string `json:"successUrl"`
		CancelURL   string `json:"cancelUrl"`
	}

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Default values
	if data.Currency == "" {
		data.Currency = "usd"
	}
	if data.SuccessURL == "" {
		data.SuccessURL = "https://your-domain.com/success"
	}
	if data.CancelURL == "" {
		data.CancelURL = "https://your-domain.com/cancel"
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(data.Currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(data.ProductName),
					},
					UnitAmount: stripe.Int64(data.Amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(data.SuccessURL),
		CancelURL:  stripe.String(data.CancelURL),
	}

	s, err := session.New(params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, CheckoutResponse{
		URL: s.URL,
		ID:  s.ID,
	})
}

// VerifyPayment verifies a payment
func (h *Handlers) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "Missing payment ID")
		return
	}

	pi, err := paymentintent.Get(id, nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":     pi.ID,
		"status": pi.Status,
		"amount": pi.Amount,
	})
}

// ListProducts lists Stripe products
func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedLimit
		}
	}

	params := &stripe.ProductListParams{
		Active: stripe.Bool(true),
	}
	params.Limit = stripe.Int64(int64(limit))

	iterator := product.List(params)
	products := []map[string]interface{}{}

	for iterator.Next() {
		p := iterator.Product()
		products = append(products, map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"description": p.Description,
			"images":      p.Images,
			"metadata":    p.Metadata,
		})
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"products": products,
	})
}

// GetProduct gets a single product by ID
func (h *Handlers) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "Missing product ID")
		return
	}

	p, err := product.Get(id, nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":          p.ID,
		"name":        p.Name,
		"description": p.Description,
		"images":      p.Images,
		"metadata":    p.Metadata,
	})
}
