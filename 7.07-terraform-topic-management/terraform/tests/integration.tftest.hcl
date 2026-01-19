# -----------------------------------------------------------------------------
# Integration Tests for Kafka Topics
# -----------------------------------------------------------------------------
# These tests run against an actual Kafka cluster to verify that topics are
# created correctly with the expected configurations.
#
# Prerequisites:
#   - A running Kafka cluster (docker compose up -d)
#   - Terraform 1.6+
#
# Run tests with: terraform test -filter=tests/integration.tftest.hcl
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Test: Apply and Verify Topic Creation
# This test actually creates topics in the Kafka cluster
# -----------------------------------------------------------------------------

run "create_topics_in_cluster" {
  command = apply

  variables {
    environment                = "integration"
    kafka_bootstrap_servers    = ["localhost:9092", "localhost:9094", "localhost:9095"]
    default_replication_factor = 3
    default_partitions         = 3
    topics                     = {}
  }

  # Verify topics were created successfully
  assert {
    condition     = kafka_topic.orders.name == "integration-orders"
    error_message = "Orders topic was not created with correct name"
  }

  assert {
    condition     = kafka_topic.payments.name == "integration-payments"
    error_message = "Payments topic was not created with correct name"
  }

  assert {
    condition     = kafka_topic.customers.name == "integration-customers"
    error_message = "Customers topic was not created with correct name"
  }

  assert {
    condition     = output.topic_count == 7
    error_message = "Expected 7 topics to be created"
  }
}

# -----------------------------------------------------------------------------
# Test: Verify Replication Factor
# Ensures topics are replicated correctly across brokers
# -----------------------------------------------------------------------------

run "verify_replication" {
  command = apply

  variables {
    environment                = "repl-test"
    kafka_bootstrap_servers    = ["localhost:9092", "localhost:9094", "localhost:9095"]
    default_replication_factor = 3
    default_partitions         = 3
    topics                     = {}
  }

  # All topics should have replication factor of 3
  assert {
    condition     = kafka_topic.orders.replication_factor == 3
    error_message = "Orders topic should have replication factor of 3"
  }

  assert {
    condition     = kafka_topic.dlq_orders.replication_factor == 3
    error_message = "DLQ topics should also have replication factor of 3"
  }
}

# -----------------------------------------------------------------------------
# Test: Dynamic Topic Creation
# Verifies that topics defined in var.topics are created correctly
# -----------------------------------------------------------------------------

run "create_dynamic_topics" {
  command = apply

  variables {
    environment             = "dynamic"
    kafka_bootstrap_servers = ["localhost:9092", "localhost:9094", "localhost:9095"]
    topics = {
      "test-events" = {
        partitions         = 4
        replication_factor = 3
        config = {
          "cleanup.policy" = "delete"
          "retention.ms"   = "3600000"
        }
      }
      "test-state" = {
        partitions = 2
        config = {
          "cleanup.policy" = "compact"
        }
      }
    }
  }

  # Verify dynamic topics are created
  assert {
    condition     = kafka_topic.dynamic["test-events"].name == "dynamic-test-events"
    error_message = "Dynamic test-events topic was not created correctly"
  }

  assert {
    condition     = kafka_topic.dynamic["test-events"].partitions == 4
    error_message = "Dynamic test-events topic should have 4 partitions"
  }

  assert {
    condition     = kafka_topic.dynamic["test-state"].name == "dynamic-test-state"
    error_message = "Dynamic test-state topic was not created correctly"
  }

  assert {
    condition     = kafka_topic.dynamic["test-state"].partitions == 2
    error_message = "Dynamic test-state topic should have 2 partitions"
  }

  # Verify total topic count includes dynamic topics
  assert {
    condition     = output.topic_count == 9
    error_message = "Should have 9 topics (7 core + 2 dynamic)"
  }
}

# -----------------------------------------------------------------------------
# Test: Update Topic Configuration
# Verifies that topic configurations can be updated
# -----------------------------------------------------------------------------

run "initial_topic_state" {
  command = apply

  variables {
    environment             = "update"
    kafka_bootstrap_servers = ["localhost:9092", "localhost:9094", "localhost:9095"]
    topics = {
      "mutable-config" = {
        partitions = 3
        config = {
          "retention.ms" = "86400000"
        }
      }
    }
  }

  assert {
    condition     = kafka_topic.dynamic["mutable-config"].config["retention.ms"] == "86400000"
    error_message = "Initial retention should be 1 day"
  }
}

run "update_topic_retention" {
  command = apply

  variables {
    environment             = "update"
    kafka_bootstrap_servers = ["localhost:9092", "localhost:9094", "localhost:9095"]
    topics = {
      "mutable-config" = {
        partitions = 3
        config = {
          "retention.ms" = "172800000"
        }
      }
    }
  }

  assert {
    condition     = kafka_topic.dynamic["mutable-config"].config["retention.ms"] == "172800000"
    error_message = "Updated retention should be 2 days"
  }
}

# -----------------------------------------------------------------------------
# Test: Idempotency
# Verifies that running apply multiple times produces the same result
# -----------------------------------------------------------------------------

run "idempotency_first_apply" {
  command = apply

  variables {
    environment             = "idempotent"
    kafka_bootstrap_servers = ["localhost:9092", "localhost:9094", "localhost:9095"]
    topics                  = {}
  }

  assert {
    condition     = output.topic_count == 7
    error_message = "First apply should create 7 topics"
  }
}

run "idempotency_second_apply" {
  command = apply

  variables {
    environment             = "idempotent"
    kafka_bootstrap_servers = ["localhost:9092", "localhost:9094", "localhost:9095"]
    topics                  = {}
  }

  # Second apply should result in the same state
  assert {
    condition     = output.topic_count == 7
    error_message = "Second apply should still have 7 topics (idempotent)"
  }
}

# -----------------------------------------------------------------------------
# Test: Cleanup Policy Verification
# Verifies compacted vs delete cleanup policies work correctly
# -----------------------------------------------------------------------------

run "verify_cleanup_policies" {
  command = apply

  variables {
    environment             = "cleanup"
    kafka_bootstrap_servers = ["localhost:9092", "localhost:9094", "localhost:9095"]
    topics                  = {}
  }

  # Event topics should use delete policy
  assert {
    condition     = kafka_topic.orders.config["cleanup.policy"] == "delete"
    error_message = "Orders topic should use delete cleanup policy"
  }

  # State topics should use compact policy
  assert {
    condition     = kafka_topic.customers.config["cleanup.policy"] == "compact"
    error_message = "Customers topic should use compact cleanup policy"
  }

  assert {
    condition     = kafka_topic.products.config["cleanup.policy"] == "compact"
    error_message = "Products topic should use compact cleanup policy"
  }
}

# -----------------------------------------------------------------------------
# Test: All Outputs Are Populated
# Verifies that all expected outputs are available after apply
# -----------------------------------------------------------------------------

run "verify_all_outputs" {
  command = apply

  variables {
    environment             = "outputs"
    kafka_bootstrap_servers = ["localhost:9092", "localhost:9094", "localhost:9095"]
    topics                  = {}
  }

  # Verify individual topic outputs
  assert {
    condition     = output.orders_topic.name == "outputs-orders"
    error_message = "Orders topic output should be populated"
  }

  assert {
    condition     = output.payments_topic.name == "outputs-payments"
    error_message = "Payments topic output should be populated"
  }

  assert {
    condition     = output.customers_topic.name == "outputs-customers"
    error_message = "Customers topic output should be populated"
  }

  # Verify DLQ outputs
  assert {
    condition     = output.dlq_topics.orders == "outputs-dlq-orders"
    error_message = "DLQ orders output should be populated"
  }

  # Verify summary outputs
  assert {
    condition     = output.environment == "outputs"
    error_message = "Environment output should match input"
  }

  assert {
    condition     = length(output.all_topic_names) == 7
    error_message = "All topic names output should contain 7 topics"
  }
}
