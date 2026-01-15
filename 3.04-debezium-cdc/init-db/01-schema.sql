-- E-commerce Database Schema for Debezium CDC Exercise
-- =====================================================

-- Enable the pgoutput plugin for logical replication (Debezium default)
-- This is already available in PostgreSQL 10+

-- Create schema
CREATE SCHEMA IF NOT EXISTS shop;

-- Customers table
CREATE TABLE shop.customers (
    customer_id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Products table
CREATE TABLE shop.products (
    product_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    category VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Orders table
CREATE TABLE shop.orders (
    order_id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES shop.customers(customer_id),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    total_amount DECIMAL(10, 2) NOT NULL DEFAULT 0,
    shipping_address TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Order items table
CREATE TABLE shop.order_items (
    order_item_id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES shop.orders(order_id),
    product_id INTEGER NOT NULL REFERENCES shop.products(product_id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_orders_customer_id ON shop.orders(customer_id);
CREATE INDEX idx_orders_status ON shop.orders(status);
CREATE INDEX idx_order_items_order_id ON shop.order_items(order_id);
CREATE INDEX idx_order_items_product_id ON shop.order_items(product_id);

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION shop.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to auto-update updated_at
CREATE TRIGGER update_customers_updated_at
    BEFORE UPDATE ON shop.customers
    FOR EACH ROW
    EXECUTE FUNCTION shop.update_updated_at_column();

CREATE TRIGGER update_products_updated_at
    BEFORE UPDATE ON shop.products
    FOR EACH ROW
    EXECUTE FUNCTION shop.update_updated_at_column();

CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON shop.orders
    FOR EACH ROW
    EXECUTE FUNCTION shop.update_updated_at_column();

-- Insert sample customers
INSERT INTO shop.customers (email, first_name, last_name) VALUES
    ('alice@example.com', 'Alice', 'Johnson'),
    ('bob@example.com', 'Bob', 'Smith'),
    ('charlie@example.com', 'Charlie', 'Brown'),
    ('diana@example.com', 'Diana', 'Williams'),
    ('eve@example.com', 'Eve', 'Davis');

-- Insert sample products
INSERT INTO shop.products (name, description, price, stock_quantity, category) VALUES
    ('Laptop Pro 15"', 'High-performance laptop with 16GB RAM', 1299.99, 50, 'Electronics'),
    ('Wireless Mouse', 'Ergonomic wireless mouse with USB receiver', 29.99, 200, 'Electronics'),
    ('Mechanical Keyboard', 'RGB mechanical keyboard with Cherry MX switches', 149.99, 75, 'Electronics'),
    ('USB-C Hub', '7-in-1 USB-C hub with HDMI and SD card reader', 49.99, 150, 'Electronics'),
    ('Monitor 27"', '4K IPS monitor with HDR support', 399.99, 30, 'Electronics'),
    ('Webcam HD', '1080p webcam with built-in microphone', 79.99, 100, 'Electronics'),
    ('Laptop Stand', 'Adjustable aluminum laptop stand', 39.99, 120, 'Accessories'),
    ('Desk Mat', 'Large desk mat with anti-slip base', 24.99, 200, 'Accessories'),
    ('Cable Management Kit', 'Complete cable organization solution', 19.99, 300, 'Accessories'),
    ('Headphone Stand', 'Wooden headphone stand with cable holder', 34.99, 80, 'Accessories');

-- Insert sample orders
INSERT INTO shop.orders (customer_id, status, total_amount, shipping_address) VALUES
    (1, 'COMPLETED', 1349.98, '123 Main St, New York, NY 10001'),
    (2, 'SHIPPED', 179.98, '456 Oak Ave, Los Angeles, CA 90001'),
    (1, 'PENDING', 449.98, '123 Main St, New York, NY 10001'),
    (3, 'PROCESSING', 29.99, '789 Pine Rd, Chicago, IL 60601');

-- Insert sample order items
INSERT INTO shop.order_items (order_id, product_id, quantity, unit_price) VALUES
    (1, 1, 1, 1299.99),  -- Laptop Pro
    (1, 2, 1, 29.99),    -- Wireless Mouse
    (1, 9, 1, 19.99),    -- Cable Management Kit
    (2, 3, 1, 149.99),   -- Mechanical Keyboard
    (2, 2, 1, 29.99),    -- Wireless Mouse
    (3, 5, 1, 399.99),   -- Monitor
    (3, 4, 1, 49.99),    -- USB-C Hub
    (4, 2, 1, 29.99);    -- Wireless Mouse

-- Create a publication for Debezium
-- This tells PostgreSQL which tables to include in logical replication
CREATE PUBLICATION debezium_publication FOR TABLE
    shop.customers,
    shop.products,
    shop.orders,
    shop.order_items;

-- Grant permissions (Debezium needs replication permissions)
-- Note: The postgres user already has these permissions

-- Verify setup
SELECT 'Database initialized successfully!' AS status;
