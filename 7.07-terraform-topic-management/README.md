# Exercise 7.07: Terraform Topic Management

## Learning Objectives

- Understand Infrastructure-as-Code (IaC) principles for Kafka topic management
- Use the Terraform Kafka provider to create and configure topics
- Implement topic naming conventions and configuration standards
- Write and run Terraform tests to validate configurations
- Apply GitOps practices for Kafka infrastructure

## Background

Managing Kafka topics manually through CLI commands doesn't scale well in production environments. As your Kafka deployment grows, you need:

- **Version Control**: Track changes to topic configurations over time
- **Consistency**: Ensure topics are configured the same way across environments
- **Automation**: Create topics as part of CI/CD pipelines
- **Compliance**: Enforce organizational standards for topic configurations
- **Collaboration**: Enable team review of infrastructure changes

Terraform provides a declarative approach to managing Kafka resources, treating your topic configurations as code that can be versioned, reviewed, and tested.

### The Mongey/kafka Provider

This exercise uses the [Mongey/kafka](https://registry.terraform.io/providers/Mongey/kafka/latest) Terraform provider, which supports:

- **Topics**: Create, update, and delete topics with full configuration control
- **ACLs**: Manage access control lists (covered in Exercise 5.01)
- **Quotas**: Set client quotas (covered in Exercise 7.02)

### Terraform Testing

Terraform 1.6+ includes a built-in testing framework that allows you to:

- Validate configurations without applying changes (`command = plan`)
- Test actual infrastructure creation (`command = apply`)
- Assert expected values and behaviors
- Run integration tests against real infrastructure

## Prerequisites

- Docker and Docker Compose
- Terraform >= 1.6.0 (for testing support)
- Basic understanding of Terraform concepts (providers, resources, variables)
- Completed exercises: 1.02 (Topic Creation), 7.06 (Dynamic Broker Config)

### Installing Terraform

**macOS (Homebrew):**
```bash
brew tap hashicorp/tap
brew install hashicorp/tap/terraform
```

**Linux:**
```bash
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform
```

**Verify installation:**
```bash
terraform version
```

## Setup

1. Start the Kafka cluster:
```bash
docker compose up -d
```

2. Wait for the cluster to be healthy:
```bash
docker compose ps
```

3. Navigate to the Terraform directory:
```bash
cd terraform
```

4. Initialize Terraform:
```bash
terraform init
```

You should see:
```
Initializing the backend...
Initializing provider plugins...
- Finding mongey/kafka versions matching "~> 0.7"...
- Installing mongey/kafka v0.7.1...
Terraform has been successfully initialized!
```

## Project Structure

```
7.07-terraform-topic-management/
├── docker-compose.yaml          # Kafka cluster
├── README.md                    # This file
├── scripts/
│   ├── verify-topics.sh         # Verify topics in Kafka
│   └── cleanup-test-topics.sh   # Remove test topics
└── terraform/
    ├── providers.tf             # Provider configuration
    ├── variables.tf             # Input variables
    ├── main.tf                  # Topic definitions
    ├── outputs.tf               # Output values
    ├── terraform.tfvars.example # Example variable values
    ├── .gitignore               # Ignore state files
    └── tests/
        ├── topics.tftest.hcl        # Unit tests (plan only)
        └── integration.tftest.hcl   # Integration tests (apply)
```

## Tasks

### Task 1: Explore the Terraform Configuration

Examine the main configuration file:

```bash
cat main.tf
```

**Key observations:**

1. **Environment Prefix**: All topics use `${var.environment}` prefix for isolation
2. **Explicit Configurations**: Each topic has specific settings for its use case
3. **Cleanup Policies**: Event topics use `delete`, state topics use `compact`
4. **Min Insync Replicas**: All topics require 2 replicas for durability

**Question**: Why do DLQ (Dead Letter Queue) topics have only 1 partition?

### Task 2: Create the tfvars File

Copy the example variables file:

```bash
cp terraform.tfvars.example terraform.tfvars
```

Review the example file - it includes both the core topics and some dynamic topics:

```bash
cat terraform.tfvars.example
```

For a minimal setup with just core topics, edit `terraform.tfvars`:

```hcl
kafka_bootstrap_servers = ["localhost:9092", "localhost:9094", "localhost:9095"]
environment = "dev"
default_replication_factor = 3
default_partitions = 3
topics = {}  # Empty = only core topics
```

Or keep the example dynamic topics for a fuller demonstration.

### Task 3: Plan the Infrastructure

Run Terraform plan to see what will be created:

```bash
terraform plan
```

Expected output (with empty `topics = {}`):
```
Plan: 7 to add, 0 to change, 0 to destroy.

Changes to Outputs:
  + all_topic_names   = [
      + "dev-orders",
      + "dev-order-events",
      + "dev-payments",
      + "dev-customers",
      + "dev-products",
      + "dev-dlq-orders",
      + "dev-dlq-payments",
    ]
  + topic_count       = 7
```

If using the example tfvars (with dynamic topics), you'll see 11 topics instead.

**Question**: What happens if you change `environment = "staging"`? How many resources need to change?

### Task 4: Apply the Configuration

Create the topics in Kafka:

```bash
terraform apply
```

Type `yes` when prompted.

Verify the topics exist:

```bash
# Using the verification script
chmod +x ../scripts/verify-topics.sh
../scripts/verify-topics.sh dev

# Or directly with Kafka CLI
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --list | grep "^dev-"
```

### Task 5: Examine Topic Configurations

Check the configuration of a compacted topic:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --entity-type topics \
  --entity-name dev-customers \
  --describe
```

You should see:
```
Dynamic configs for topic dev-customers are:
  cleanup.policy=compact sensitive=false
  min.cleanable.dirty.ratio=0.5 sensitive=false
  delete.retention.ms=86400000 sensitive=false
  min.insync.replicas=2 sensitive=false
```

### Task 6: Run Terraform Tests (Plan-Only)

Run the unit tests that validate configuration without creating resources:

```bash
terraform test -filter=tests/topics.tftest.hcl
```

Expected output:
```
tests/topics.tftest.hcl... in progress
  run "validate_default_variables"... pass
  run "validate_topic_naming"... pass
  run "validate_core_topic_config"... pass
  run "validate_compacted_topics"... pass
  run "validate_dlq_topics"... pass
  run "validate_min_insync_replicas"... pass
  run "validate_dynamic_topics"... pass
  run "validate_topic_count"... pass
  run "validate_prod_environment"... pass
  run "validate_retention_config"... pass
tests/topics.tftest.hcl... pass

Success! 10 passed, 0 failed.
```

### Task 7: Run Integration Tests

Run tests that create actual topics in the cluster:

```bash
terraform test -filter=tests/integration.tftest.hcl
```

These tests:
1. Create topics with different environment prefixes
2. Verify replication factors
3. Test dynamic topic creation
4. Validate configuration updates
5. Check idempotency (running apply twice produces same result)

**Note**: Integration tests create real topics. Use the cleanup script afterwards:

```bash
chmod +x ../scripts/cleanup-test-topics.sh
../scripts/cleanup-test-topics.sh
```

### Task 8: Add Dynamic Topics

Edit `terraform.tfvars` to add custom topics:

```hcl
topics = {
  "analytics-events" = {
    partitions = 6
    config = {
      "cleanup.policy" = "delete"
      "retention.ms"   = "86400000"
    }
  }

  "user-sessions" = {
    partitions = 12
    config = {
      "cleanup.policy" = "compact"
    }
  }
}
```

Plan and apply the changes:

```bash
terraform plan
terraform apply
```

Verify the new topics:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --topic dev-analytics-events
```

### Task 9: Modify Topic Configuration

Change the retention for the analytics topic:

```hcl
topics = {
  "analytics-events" = {
    partitions = 6
    config = {
      "cleanup.policy" = "delete"
      "retention.ms"   = "172800000"  # Changed from 1 day to 2 days
    }
  }
  # ... rest unchanged
}
```

Apply and observe:

```bash
terraform plan
```

Notice that Terraform shows an in-place update, not a replacement:
```
~ resource "kafka_topic" "dynamic" {
    ~ config = {
        ~ "retention.ms" = "86400000" -> "172800000"
      }
  }
```

### Task 10: Use Terraform State Commands

Explore the Terraform state:

```bash
# List all managed resources
terraform state list

# Show details of a specific topic
terraform state show kafka_topic.orders

# View outputs
terraform output
terraform output -json all_topic_names
```

### Task 11: Import Existing Topic

If you have an existing topic not managed by Terraform, you can import it:

```bash
# First, create a topic manually
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --create \
  --topic dev-legacy-events \
  --partitions 3 \
  --replication-factor 3
```

Add the topic to your configuration (in `main.tf` or via `var.topics`):

```hcl
# In terraform.tfvars
topics = {
  "legacy-events" = {
    partitions = 3
    replication_factor = 3
    config = {}
  }
  # ... other topics
}
```

Import the topic into Terraform state:

```bash
terraform import 'kafka_topic.dynamic["legacy-events"]' dev-legacy-events
```

Now Terraform manages this topic!

### Task 12: View in Kafka UI

Open Kafka UI to visualize the topics:

```bash
open http://localhost:8080
```

Navigate to **Topics** and observe:
- All topics with `dev-` prefix
- Partition distribution across brokers
- Topic configurations

### Task 13: Simulate CI/CD Workflow

In a real CI/CD pipeline, you would:

1. **Validate** the configuration:
```bash
terraform validate
```

2. **Format** the code:
```bash
terraform fmt -check
```

3. **Plan** with output file:
```bash
terraform plan -out=tfplan
```

4. **Apply** the plan file (no prompt):
```bash
terraform apply tfplan
```

### Task 14: Multi-Environment Setup

Create separate variable files for each environment:

```bash
# Create staging variables
cat > terraform.staging.tfvars << 'EOF'
environment = "staging"
kafka_bootstrap_servers = ["localhost:9092", "localhost:9094", "localhost:9095"]
default_replication_factor = 3
default_partitions = 3
topics = {}
EOF
```

Apply to staging:

```bash
terraform plan -var-file=terraform.staging.tfvars
terraform apply -var-file=terraform.staging.tfvars
```

Now you have both `dev-*` and `staging-*` topics!

## Verification

You've successfully completed this exercise when you can:

- ✅ Initialize Terraform with the Kafka provider
- ✅ Create topics using `terraform apply`
- ✅ Run plan-only tests with `terraform test`
- ✅ Run integration tests against the real cluster
- ✅ Add dynamic topics through variables
- ✅ Modify topic configurations
- ✅ Import existing topics into Terraform state
- ✅ Understand the CI/CD workflow for topic management

## Key Concepts

### Why Infrastructure-as-Code for Kafka?

| Approach | Pros | Cons |
|----------|------|------|
| Manual CLI | Quick for testing | No audit trail, inconsistent |
| Scripts | Automated | Hard to maintain, no state |
| Terraform | Declarative, versioned, tested | Learning curve |

### Topic Configuration Best Practices

1. **Naming Convention**: Use environment prefix (`dev-`, `staging-`, `prod-`)
2. **Replication**: Always use RF >= 3 for production
3. **Min ISR**: Set to `replication_factor - 1` for durability
4. **Cleanup Policy**: Use `compact` for state, `delete` for events
5. **Retention**: Match business requirements (shorter for events, longer for audit)

### Terraform Testing Strategies

| Test Type | Command | Use Case |
|-----------|---------|----------|
| Validate | `terraform validate` | Syntax checking |
| Plan | `terraform test` (command=plan) | Configuration validation |
| Apply | `terraform test` (command=apply) | Integration testing |
| Format | `terraform fmt -check` | Code style |

## Cleanup

1. Destroy Terraform-managed topics:
```bash
cd terraform
terraform destroy
```

2. Clean up test topics:
```bash
../scripts/cleanup-test-topics.sh
```

3. Stop the Kafka cluster:
```bash
cd ..
docker compose down -v
```

## Troubleshooting

### Provider Authentication Errors

If you see connection errors:

```bash
# Verify Kafka is running
docker compose ps

# Check broker connectivity
docker exec kafka-1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server localhost:9092
```

### State Lock Issues

If Terraform state is locked:

```bash
# Force unlock (use with caution)
terraform force-unlock LOCK_ID
```

### Topic Already Exists

If a topic exists but isn't in Terraform state:

```bash
# Import it
terraform import kafka_topic.orders dev-orders

# Or delete and recreate
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --delete --topic dev-orders
terraform apply
```

### Test Failures

If tests fail:

```bash
# Run with verbose output
terraform test -verbose

# Run specific test
terraform test -filter=tests/topics.tftest.hcl -run=validate_topic_naming
```

## Best Practices

1. **State Management**: Use remote state (S3, GCS, Terraform Cloud) in production
2. **Workspace Isolation**: Use Terraform workspaces or separate state files per environment
3. **Code Review**: Require PR reviews for topic changes
4. **Drift Detection**: Run `terraform plan` regularly to detect manual changes
5. **Documentation**: Comment topic configurations with their purpose
6. **Testing**: Always run tests before merging changes

## Additional Resources

- [Mongey/kafka Provider Documentation](https://registry.terraform.io/providers/Mongey/kafka/latest/docs)
- [Terraform Testing Framework](https://developer.hashicorp.com/terraform/language/tests)
- [Kafka Topic Configuration Reference](https://kafka.apache.org/documentation/#topicconfigs)
- [GitOps for Kafka](https://www.confluent.io/blog/gitops-for-apache-kafka-with-confluent/)

## Next Steps

Continue to [Exercise 8.01: MirrorMaker Backup](../8.01-mirrormaker-backup/) to learn about cross-cluster replication, or explore:

- [Exercise 5.01: Access Control Lists](../5.01-access-control-lists/) - Add ACL management to Terraform
- [Exercise 7.02: Client Quotas](../7.02-client-quotas/) - Manage quotas with Terraform