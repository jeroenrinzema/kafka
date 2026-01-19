terraform {
  required_version = ">= 1.6.0"

  required_providers {
    kafka = {
      source  = "Mongey/kafka"
      version = "~> 0.7"
    }
  }
}

provider "kafka" {
  bootstrap_servers = var.kafka_bootstrap_servers

  # TLS configuration (disabled for local development)
  skip_tls_verify = true
  tls_enabled     = false

  # SASL configuration (disabled for local development)
  sasl_mechanism = "plain"
  sasl_username  = ""
  sasl_password  = ""

  # Timeout settings
  timeout = 30
}
