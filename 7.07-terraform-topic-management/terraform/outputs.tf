# -----------------------------------------------------------------------------
# Outputs
# -----------------------------------------------------------------------------
# These outputs provide information about the created Kafka topics that can be
# used by other Terraform modules or for debugging/verification purposes.
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Core Topic Outputs
# -----------------------------------------------------------------------------

output "orders_topic" {
  description = "Orders topic details"
  value = {
    name               = kafka_topic.orders.name
    partitions         = kafka_topic.orders.partitions
    replication_factor = kafka_topic.orders.replication_factor
  }
}

output "order_events_topic" {
  description = "Order events topic details"
  value = {
    name               = kafka_topic.order_events.name
    partitions         = kafka_topic.order_events.partitions
    replication_factor = kafka_topic.order_events.replication_factor
  }
}

output "payments_topic" {
  description = "Payments topic details"
  value = {
    name               = kafka_topic.payments.name
    partitions         = kafka_topic.payments.partitions
    replication_factor = kafka_topic.payments.replication_factor
  }
}

# -----------------------------------------------------------------------------
# Compacted Topic Outputs
# -----------------------------------------------------------------------------

output "customers_topic" {
  description = "Customers (compacted) topic details"
  value = {
    name               = kafka_topic.customers.name
    partitions         = kafka_topic.customers.partitions
    replication_factor = kafka_topic.customers.replication_factor
    cleanup_policy     = "compact"
  }
}

output "products_topic" {
  description = "Products (compacted) topic details"
  value = {
    name               = kafka_topic.products.name
    partitions         = kafka_topic.products.partitions
    replication_factor = kafka_topic.products.replication_factor
    cleanup_policy     = "compact"
  }
}

# -----------------------------------------------------------------------------
# DLQ Topic Outputs
# -----------------------------------------------------------------------------

output "dlq_topics" {
  description = "Dead Letter Queue topic names"
  value = {
    orders   = kafka_topic.dlq_orders.name
    payments = kafka_topic.dlq_payments.name
  }
}

# -----------------------------------------------------------------------------
# Dynamic Topic Outputs
# -----------------------------------------------------------------------------

output "dynamic_topics" {
  description = "Dynamically created topics from var.topics"
  value = {
    for name, topic in kafka_topic.dynamic : name => {
      name               = topic.name
      partitions         = topic.partitions
      replication_factor = topic.replication_factor
    }
  }
}

# -----------------------------------------------------------------------------
# Summary Outputs
# -----------------------------------------------------------------------------

output "all_topic_names" {
  description = "List of all managed topic names"
  value       = local.all_topic_names
}

output "topic_count" {
  description = "Total number of topics managed by Terraform"
  value       = length(local.all_topic_names)
}

output "environment" {
  description = "Environment name used as topic prefix"
  value       = var.environment
}

output "bootstrap_servers" {
  description = "Kafka bootstrap servers used"
  value       = var.kafka_bootstrap_servers
}
