# -----------------------------------------------------------------------------
# Terraform Tests for Kafka Topics
# -----------------------------------------------------------------------------
# These tests validate the Kafka topic configurations using Terraform's built-in
# testing framework (terraform test command, available in Terraform 1.6+).
#
# Run tests with: terraform test
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Test: Default Configuration Values
# Validates that default variable values are set correctly
# -----------------------------------------------------------------------------

run "validate_default_variables" {
  command = plan

  variables {
    environment = "test"
  }

  # Verify default partitions
  assert {
    condition     = var.default_partitions == 3
    error_message = "Default partitions should be 3"
  }

  # Verify default replication factor
  assert {
    condition     = var.default_replication_factor == 3
    error_message = "Default replication factor should be 3"
  }
}

# -----------------------------------------------------------------------------
# Test: Topic Naming Convention
# Validates that topics follow the environment prefix naming convention
# -----------------------------------------------------------------------------

run "validate_topic_naming" {
  command = plan

  variables {
    environment = "staging"
  }

  # Verify orders topic name includes environment prefix
  assert {
    condition     = kafka_topic.orders.name == "staging-orders"
    error_message = "Orders topic should have environment prefix: got ${kafka_topic.orders.name}"
  }

  # Verify payments topic name includes environment prefix
  assert {
    condition     = kafka_topic.payments.name == "staging-payments"
    error_message = "Payments topic should have environment prefix"
  }

  # Verify customers topic name includes environment prefix
  assert {
    condition     = kafka_topic.customers.name == "staging-customers"
    error_message = "Customers topic should have environment prefix"
  }
}

# -----------------------------------------------------------------------------
# Test: Core Topics Configuration
# Validates that core event topics have correct configurations
# -----------------------------------------------------------------------------

run "validate_core_topic_config" {
  command = plan

  variables {
    environment = "test"
  }

  # Orders topic should have correct partitions
  assert {
    condition     = kafka_topic.orders.partitions == 3
    error_message = "Orders topic should have 3 partitions"
  }

  # Orders topic should have correct replication factor
  assert {
    condition     = kafka_topic.orders.replication_factor == 3
    error_message = "Orders topic should have replication factor of 3"
  }

  # Order events topic should have 6 partitions (higher throughput)
  assert {
    condition     = kafka_topic.order_events.partitions == 6
    error_message = "Order events topic should have 6 partitions for higher throughput"
  }
}

# -----------------------------------------------------------------------------
# Test: Compacted Topics Configuration
# Validates that compacted topics have correct cleanup policy
# -----------------------------------------------------------------------------

run "validate_compacted_topics" {
  command = plan

  variables {
    environment = "test"
  }

  # Customers topic should use compaction
  assert {
    condition     = kafka_topic.customers.config["cleanup.policy"] == "compact"
    error_message = "Customers topic should use log compaction"
  }

  # Products topic should use compaction
  assert {
    condition     = kafka_topic.products.config["cleanup.policy"] == "compact"
    error_message = "Products topic should use log compaction"
  }
}

# -----------------------------------------------------------------------------
# Test: DLQ Topics Configuration
# Validates Dead Letter Queue topics have correct settings
# -----------------------------------------------------------------------------

run "validate_dlq_topics" {
  command = plan

  variables {
    environment = "test"
  }

  # DLQ topics should have 1 partition (low throughput, ordering)
  assert {
    condition     = kafka_topic.dlq_orders.partitions == 1
    error_message = "DLQ orders topic should have 1 partition"
  }

  assert {
    condition     = kafka_topic.dlq_payments.partitions == 1
    error_message = "DLQ payments topic should have 1 partition"
  }

  # DLQ topics should have delete cleanup policy
  assert {
    condition     = kafka_topic.dlq_orders.config["cleanup.policy"] == "delete"
    error_message = "DLQ topics should use delete cleanup policy"
  }
}

# -----------------------------------------------------------------------------
# Test: Min Insync Replicas
# Validates that all topics have min.insync.replicas set for durability
# -----------------------------------------------------------------------------

run "validate_min_insync_replicas" {
  command = plan

  variables {
    environment = "test"
  }

  # All core topics should have min.insync.replicas = 2
  assert {
    condition     = kafka_topic.orders.config["min.insync.replicas"] == "2"
    error_message = "Orders topic should have min.insync.replicas = 2"
  }

  assert {
    condition     = kafka_topic.payments.config["min.insync.replicas"] == "2"
    error_message = "Payments topic should have min.insync.replicas = 2"
  }

  assert {
    condition     = kafka_topic.customers.config["min.insync.replicas"] == "2"
    error_message = "Customers topic should have min.insync.replicas = 2"
  }
}

# -----------------------------------------------------------------------------
# Test: Dynamic Topics
# Validates that dynamic topics are created correctly from var.topics
# -----------------------------------------------------------------------------

run "validate_dynamic_topics" {
  command = plan

  variables {
    environment = "test"
    topics = {
      "custom-events" = {
        partitions = 6
        config = {
          "retention.ms" = "86400000"
        }
      }
    }
  }

  # Dynamic topic should be created
  assert {
    condition     = kafka_topic.dynamic["custom-events"].name == "test-custom-events"
    error_message = "Dynamic topic should be created with environment prefix"
  }

  # Dynamic topic should have specified partitions
  assert {
    condition     = kafka_topic.dynamic["custom-events"].partitions == 6
    error_message = "Dynamic topic should have 6 partitions as specified"
  }

  # Dynamic topic should inherit default replication factor
  assert {
    condition     = kafka_topic.dynamic["custom-events"].replication_factor == 3
    error_message = "Dynamic topic should inherit default replication factor"
  }
}

# -----------------------------------------------------------------------------
# Test: Topic Count Output
# Validates that the topic count output is correct
# -----------------------------------------------------------------------------

run "validate_topic_count" {
  command = plan

  variables {
    environment = "test"
    topics      = {}
  }

  # With no dynamic topics, we should have 7 core topics
  # (orders, order_events, payments, customers, products, dlq_orders, dlq_payments)
  assert {
    condition     = output.topic_count == 7
    error_message = "Should have 7 core topics when no dynamic topics are defined"
  }
}

# -----------------------------------------------------------------------------
# Test: Different Environments
# Validates that the same config works for different environments
# -----------------------------------------------------------------------------

run "validate_prod_environment" {
  command = plan

  variables {
    environment                = "prod"
    default_replication_factor = 3
    default_partitions         = 6
  }

  # Prod environment should use prod prefix
  assert {
    condition     = kafka_topic.orders.name == "prod-orders"
    error_message = "Prod orders topic should have prod prefix"
  }

  # Prod can have different partition count
  assert {
    condition     = kafka_topic.orders.partitions == 6
    error_message = "Prod orders topic should have 6 partitions"
  }
}

# -----------------------------------------------------------------------------
# Test: Retention Configuration
# Validates retention settings are applied correctly
# -----------------------------------------------------------------------------

run "validate_retention_config" {
  command = plan

  variables {
    environment = "test"
  }

  # Orders topic should have 7 days retention (604800000ms)
  assert {
    condition     = kafka_topic.orders.config["retention.ms"] == "604800000"
    error_message = "Orders topic should have 7 days retention"
  }

  # Order events should have 30 days retention (2592000000ms)
  assert {
    condition     = kafka_topic.order_events.config["retention.ms"] == "2592000000"
    error_message = "Order events topic should have 30 days retention"
  }

  # DLQ topics should have 30 days retention for investigation
  assert {
    condition     = kafka_topic.dlq_orders.config["retention.ms"] == "2592000000"
    error_message = "DLQ topics should have 30 days retention for investigation"
  }
}
