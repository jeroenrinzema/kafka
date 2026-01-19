# -----------------------------------------------------------------------------
# Kafka Topics - Main Configuration
# -----------------------------------------------------------------------------
# This file defines Kafka topics managed via Terraform using the Mongey/kafka
# provider. Topics are defined either directly or through the var.topics map
# for dynamic configuration.
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Core Event Topics
# These topics are essential for the platform's event-driven architecture
# -----------------------------------------------------------------------------

resource "kafka_topic" "orders" {
  name               = "${var.environment}-orders"
  partitions         = var.default_partitions
  replication_factor = var.default_replication_factor

  config = {
    "cleanup.policy"      = "delete"
    "retention.ms"        = "604800000" # 7 days
    "min.insync.replicas" = "2"
    "segment.bytes"       = "1073741824" # 1GB
  }
}

resource "kafka_topic" "order_events" {
  name               = "${var.environment}-order-events"
  partitions         = 6
  replication_factor = var.default_replication_factor

  config = {
    "cleanup.policy"      = "delete"
    "retention.ms"        = "2592000000" # 30 days
    "min.insync.replicas" = "2"
  }
}

resource "kafka_topic" "payments" {
  name               = "${var.environment}-payments"
  partitions         = var.default_partitions
  replication_factor = var.default_replication_factor

  config = {
    "cleanup.policy"      = "delete"
    "retention.ms"        = "2592000000" # 30 days
    "min.insync.replicas" = "2"
  }
}

# -----------------------------------------------------------------------------
# Compacted Topics (State/Entity Topics)
# These topics use log compaction to maintain latest state per key
# -----------------------------------------------------------------------------

resource "kafka_topic" "customers" {
  name               = "${var.environment}-customers"
  partitions         = var.default_partitions
  replication_factor = var.default_replication_factor

  config = {
    "cleanup.policy"            = "compact"
    "min.cleanable.dirty.ratio" = "0.5"
    "delete.retention.ms"       = "86400000" # 1 day
    "min.insync.replicas"       = "2"
    "segment.ms"                = "604800000" # 7 days
  }
}

resource "kafka_topic" "products" {
  name               = "${var.environment}-products"
  partitions         = var.default_partitions
  replication_factor = var.default_replication_factor

  config = {
    "cleanup.policy"            = "compact"
    "min.cleanable.dirty.ratio" = "0.5"
    "delete.retention.ms"       = "86400000" # 1 day
    "min.insync.replicas"       = "2"
  }
}

# -----------------------------------------------------------------------------
# Dead Letter Queue Topics
# Topics for failed message handling and reprocessing
# -----------------------------------------------------------------------------

resource "kafka_topic" "dlq_orders" {
  name               = "${var.environment}-dlq-orders"
  partitions         = 1
  replication_factor = var.default_replication_factor

  config = {
    "cleanup.policy"      = "delete"
    "retention.ms"        = "2592000000" # 30 days - longer retention for investigation
    "min.insync.replicas" = "2"
  }
}

resource "kafka_topic" "dlq_payments" {
  name               = "${var.environment}-dlq-payments"
  partitions         = 1
  replication_factor = var.default_replication_factor

  config = {
    "cleanup.policy"      = "delete"
    "retention.ms"        = "2592000000" # 30 days
    "min.insync.replicas" = "2"
  }
}

# -----------------------------------------------------------------------------
# Dynamic Topics from Variable
# Create additional topics defined in var.topics
# -----------------------------------------------------------------------------

resource "kafka_topic" "dynamic" {
  for_each = var.topics

  name               = "${var.environment}-${each.key}"
  partitions         = coalesce(each.value.partitions, var.default_partitions)
  replication_factor = coalesce(each.value.replication_factor, var.default_replication_factor)

  config = merge(
    {
      "min.insync.replicas" = "2"
    },
    each.value.config
  )
}

# -----------------------------------------------------------------------------
# Local Values for Topic Names
# Useful for referencing topics in other resources
# -----------------------------------------------------------------------------

locals {
  core_topics = {
    orders       = kafka_topic.orders.name
    order_events = kafka_topic.order_events.name
    payments     = kafka_topic.payments.name
    customers    = kafka_topic.customers.name
    products     = kafka_topic.products.name
  }

  dlq_topics = {
    orders   = kafka_topic.dlq_orders.name
    payments = kafka_topic.dlq_payments.name
  }

  all_topic_names = concat(
    values(local.core_topics),
    values(local.dlq_topics),
    [for t in kafka_topic.dynamic : t.name]
  )
}
