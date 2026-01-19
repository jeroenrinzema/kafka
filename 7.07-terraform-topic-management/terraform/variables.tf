variable "kafka_bootstrap_servers" {
  description = "List of Kafka bootstrap servers"
  type        = list(string)
  default     = ["localhost:9092", "localhost:9094", "localhost:9095"]
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "default_replication_factor" {
  description = "Default replication factor for topics"
  type        = number
  default     = 3
}

variable "default_partitions" {
  description = "Default number of partitions for topics"
  type        = number
  default     = 3
}

variable "topics" {
  description = "Map of topics to create with their configurations"
  type = map(object({
    partitions         = optional(number)
    replication_factor = optional(number)
    config             = optional(map(string), {})
  }))
  default = {}
}
