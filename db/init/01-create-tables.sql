-- db/init/01-create-tables.sql
-- Database initialization script for Stripe Payment Backend

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum types
CREATE TYPE order_status AS ENUM (
    'created',
    'pending', 
    'paid',
    'fulfilled',
    'canceled',
    'refunded'
);

CREATE TYPE payment_status AS ENUM (
    'pending',
    'succeeded',
    'failed',
    'canceled',
    'refunded'
);

CREATE TYPE payment_method AS ENUM (
    'card',
    'paypal',
    'apple_pay',
    'google_pay'
);

-- Orders table
CREATE TABLE orders (
    id VARCHAR(50) PRIMARY KEY,
    tracking_id VARCHAR(50) UNIQUE NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    customer_name VARCHAR(255),
    customer_phone VARCHAR(50),
    customer_ip_address INET,
    status order_status NOT NULL DEFAULT 'created',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    fulfilled_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for orders
CREATE INDEX idx_orders_tracking_id ON orders(tracking_id);
CREATE INDEX idx_orders_customer_email ON orders(customer_email);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at);

-- Order items table
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id VARCHAR(50) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id VARCHAR(50) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    download_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for order_items
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);

-- Payments table
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id VARCHAR(50) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    stripe_payment_intent_id VARCHAR(255),
    stripe_session_id VARCHAR(255),
    amount BIGINT NOT NULL, -- Amount in cents
    currency VARCHAR(3) NOT NULL DEFAULT 'usd',
    status payment_status NOT NULL DEFAULT 'pending',
    method payment_method DEFAULT 'card',
    processed_at TIMESTAMP WITH TIME ZONE,
    refunded_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for payments
CREATE UNIQUE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_stripe_payment_intent_id ON payments(stripe_payment_intent_id);
CREATE INDEX idx_payments_stripe_session_id ON payments(stripe_session_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_created_at ON payments(created_at);

-- Payment events table (for audit trail)
CREATE TABLE payment_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id VARCHAR(50) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    status payment_status NOT NULL,
    data JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for payment_events
CREATE INDEX idx_payment_events_order_id ON payment_events(order_id);
CREATE INDEX idx_payment_events_event_type ON payment_events(event_type);
CREATE INDEX idx_payment_events_created_at ON payment_events(created_at);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_orders_updated_at 
    BEFORE UPDATE ON orders 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_payments_updated_at 
    BEFORE UPDATE ON payments 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create views for common queries
CREATE VIEW order_summaries AS
SELECT 
    o.id,
    o.tracking_id,
    o.customer_email,
    o.status as order_status,
    o.created_at,
    p.amount,
    p.currency,
    p.status as payment_status,
    COUNT(oi.id) as item_count
FROM orders o
LEFT JOIN payments p ON o.id = p.order_id
LEFT JOIN order_items oi ON o.id = oi.order_id
GROUP BY o.id, o.tracking_id, o.customer_email, o.status, o.created_at, p.amount, p.currency, p.status;

-- Create view for payment statistics
CREATE VIEW payment_stats AS
SELECT 
    COUNT(*) as total_orders,
    COUNT(CASE WHEN o.status IN ('paid', 'fulfilled') THEN 1 END) as completed_orders,
    COUNT(CASE WHEN o.status = 'pending' THEN 1 END) as pending_orders,
    COUNT(CASE WHEN o.status = 'refunded' THEN 1 END) as refunded_orders,
    COALESCE(SUM(CASE WHEN o.status IN ('paid', 'fulfilled') THEN p.amount ELSE 0 END), 0) as total_revenue_cents,
    COALESCE(SUM(CASE WHEN DATE(o.created_at) = CURRENT_DATE AND o.status IN ('paid', 'fulfilled') THEN p.amount ELSE 0 END), 0) as revenue_today_cents,
    COALESCE(SUM(CASE WHEN DATE_TRUNC('month', o.created_at) = DATE_TRUNC('month', CURRENT_DATE) AND o.status IN ('paid', 'fulfilled') THEN p.amount ELSE 0 END), 0) as revenue_this_month_cents
FROM orders o
LEFT JOIN payments p ON o.id = p.order_id;

-- Insert some sample data for testing (optional)
-- Uncomment the lines below if you want test data

/*
INSERT INTO orders (id, tracking_id, customer_email, customer_name, status) VALUES
    ('ord_sample_1', 'TRK001', 'test1@example.com', 'Test Customer 1', 'created'),
    ('ord_sample_2', 'TRK002', 'test2@example.com', 'Test Customer 2', 'paid');

INSERT INTO order_items (order_id, product_id, product_name, file_type, price, quantity) VALUES
    ('ord_sample_1', 'prod_1', 'Writing Guide', 'PDF', 9.99, 1),
    ('ord_sample_2', 'prod_2', 'Character Development Workbook', 'PDF', 14.99, 1);

INSERT INTO payments (order_id, amount, currency, status) VALUES
    ('ord_sample_1', 999, 'usd', 'pending'),
    ('ord_sample_2', 1499, 'usd', 'succeeded');
*/

-- Grant permissions to the application user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO stripe_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO stripe_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO stripe_user;