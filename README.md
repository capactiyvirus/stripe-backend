# Stripe Payment Backend

A Go backend service for handling Stripe payments with order tracking and management.

## Quick Start

1. **Install Go** (version 1.21 or higher)

2. **Clone and setup**:
```bash
git clone <your-repo>
cd stripe-backend
```

3. **Install dependencies**:
```bash
go mod tidy
```

4. **Setup environment**:
```bash
cp .env.example .env
# Edit .env with your actual values
```

5. **Run the server**:
```bash
go run main.go
```

The server will start on `http://localhost:8080`

## Configuration

### Required Environment Variables

- `STRIPE_SECRET_KEY`: Your Stripe secret key (sk_test_... or sk_live_...)
- `STRIPE_PUBLISHABLE_KEY`: Your Stripe publishable key
- `STRIPE_WEBHOOK_SECRET`: Your Stripe webhook endpoint secret

### Optional Environment Variables

- `PORT`: Server port (default: 8080)
- `ENVIRONMENT`: development/production (default: development)
- `CORS_ALLOWED_ORIGINS`: Comma-separated list of allowed origins

## API Endpoints

### Payment Operations

- `POST /api/payments/create-order` - Create a new order with payment tracking
- `POST /api/payments/create-intent` - Create Stripe payment intent (legacy)
- `POST /api/payments/create-checkout` - Create Stripe checkout session (legacy)

### Order Management

- `GET /api/payments/status/{orderID}` - Get payment status by order ID
- `GET /api/payments/order/{orderID}` - Get full order details
- `GET /api/payments/track/{trackingID}` - Track payment by tracking ID
- `GET /api/payments/customer/{email}` - Get customer payment history

### Admin Endpoints

- `GET /api/payments/all` - Get all payments (with pagination)
- `GET /api/payments/stats` - Get payment statistics
- `POST /api/payments/fulfill/{orderID}` - Mark order as fulfilled
- `POST /api/payments/refund/{orderID}` - Process refund

### Webhooks

- `POST /api/payments/webhook` - Stripe webhook handler

### Product Management

- `GET /api/products` - List products
- `GET /api/products/{id}` - Get product details

## Creating an Order

```javascript
const response = await fetch('/api/payments/create-order', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    customer_info: {
      email: 'customer@example.com',
      name: 'John Doe'
    },
    items: [
      {
        product_id: '1',
        product_name: 'Writing Guide',
        file_type: 'PDF',
        price: 9.99,
        quantity: 1
      }
    ],
    metadata: {
      source: 'website'
    }
  })
});

const { order, client_secret } = await response.json();
```

## Stripe Webhooks Setup

1. In your Stripe Dashboard, go to Webhooks
2. Add endpoint: `https://yourdomain.com/api/payments/webhook`
3. Select these events:
   - `payment_intent.succeeded`
   - `payment_intent.payment_failed`
   - `payment_intent.canceled`
   - `checkout.session.completed`
4. Copy the webhook secret to your `.env` file

## Testing

Run tests:
```bash
go test ./tests/...
```

Run with race detection:
```bash
go test -race ./...
```

## Project Structure

```
├── config/          # Configuration management
├── handlers/        # HTTP handlers
├── models/          # Data models
├── store/           # Data storage (in-memory & PostgreSQL)
├── services/        # Business services (email, etc.)
├── tests/           # Test files
├── main.go          # Application entry point
└── go.mod           # Go module definition
```

## Production Deployment

1. Set `ENVIRONMENT=production`
2. Use your live Stripe keys
3. Configure proper CORS origins
4. Consider using PostgreSQL instead of in-memory storage
5. Add authentication middleware for admin endpoints
6. Set up proper logging and monitoring

## Common Issues

### CORS Errors
Make sure your frontend domain is in `CORS_ALLOWED_ORIGINS`

### Webhook Failures
Verify your webhook secret matches exactly from Stripe Dashboard

### Payment Intent Not Found
Check that the order was created successfully before creating the payment intent

## Support

For issues with the Stripe integration, check:
1. Stripe Dashboard logs
2. Application logs
3. Webhook delivery status in Stripe Dashboard