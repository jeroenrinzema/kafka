# E-commerce Order Processing System

An e-commerce platform experiences varying levels of transaction volumes depending on time of day and promotions. The Kafka-based messaging system should efficiently handle events related to order processing, inventory management, and customer notifications.

The system should manage the following events, with estimated volumes and sample schemas:

## Order Placed (500,000 events/day)

```json
{'order_id': string, 'customer_id': string, 'product_ids': array, 'timestamp': datetime, 'total_price': float, 'geo_region': string}
```

## Order Confirmed (450,000 events/day)

```json
{'customer_id': string, 'order_id': string, 'inventory_status': string, 'timestamp': datetime}
```

## User Interaction Events (1,000,000 events/day)

```json
{'event_id': string, 'user_id': string, 'content_id': string, 'interaction_type': string, 'timestamp': datetime}
```

## Payment Processed (400,000 events/day)
	
```json
{'customer_id': string, 'order_id': string, 'payment_status': string, 'timestamp': datetime}
```
	
## Order Shipped (400,000 events/day)

```json
{'customer_id', 'order_id': string, 'shipment_id': string, 'carrier': string, 'timestamp': datetime}
```
	
## Inventory Updated (600,000 events/day)

```json
{'product_id': string, 'inventory_level': integer, 'timestamp': datetime}
```
	
## Customer Notification Sent (800,000 events/day)

```json
{'notification_id': string, 'customer_id': string, 'message_type': string, 'timestamp': datetime}
```

