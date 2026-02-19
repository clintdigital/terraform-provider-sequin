variable "database_name" {
  description = "Name for the Sequin database connection"
  type        = string
}

variable "postgres_host" {
  description = "PostgreSQL host address"
  type        = string
}

variable "postgres_port" {
  description = "PostgreSQL port"
  type        = number
  default     = 5432
}

variable "postgres_db" {
  description = "PostgreSQL database name"
  type        = string
}

variable "postgres_user" {
  description = "PostgreSQL username"
  type        = string
}

variable "postgres_pass" {
  description = "PostgreSQL password"
  type        = string
  sensitive   = true
}

variable "postgres_ssl" {
  description = "Enable SSL"
  type        = bool
  default     = true
}

variable "replication_slots" {
  description = "Replication slot configuration"
  type = list(object({
    publication_name = string
    slot_name        = string
    status           = optional(string, "active")
  }))
  default = [{
    publication_name = "sequin_pub"
    slot_name        = "sequin_slot"
  }]
}

variable "consumers" {
  description = "Map of sink consumers to create (key = consumer name)"
  type = map(object({
    # Source — schemas: omit for all, { include = [...] } or { exclude = [...] }
    schemas = optional(object({
      include = optional(list(string))
      exclude = optional(list(string))
    }))

    # Source — tables: omit for all, { include = [...] } or { exclude = [...] }
    tables = optional(object({
      include = optional(list(object({
        name               = string
        group_column_names = optional(list(string))
      })))
      exclude = optional(list(object({
        name = string
      })))
    }))

    # Filters
    actions         = optional(list(string), ["insert", "update", "delete"])
    filter_function = optional(string)

    # Functions
    enrichment_function = optional(string)
    transform_function  = optional(string)
    routing_function    = optional(string)

    # Destination
    destination = object({
      type                  = string
      hosts                 = optional(string)
      topic                 = optional(string)
      tls                   = optional(bool)
      username              = optional(string)
      password              = optional(string)
      sasl_mechanism        = optional(string)
      aws_region            = optional(string)
      aws_access_key_id     = optional(string)
      aws_secret_access_key = optional(string)
      queue_url             = optional(string)
      region                = optional(string)
      access_key_id         = optional(string)
      secret_access_key     = optional(string)
      is_fifo               = optional(bool)
      stream_arn            = optional(string)
      http_endpoint         = optional(string)
      http_endpoint_path    = optional(string)
      batch                 = optional(bool)
    })

    # Advanced
    status               = optional(string)
    batch_size           = optional(number)
    message_grouping     = optional(bool)
    max_retry_count      = optional(number)
    load_shedding_policy = optional(string)
    timestamp_format     = optional(string)
  }))
  default = {}
}

variable "backfills" {
  description = "Map of backfills to create"
  type = map(object({
    consumer_name = string
    table         = optional(string)
    state         = optional(string)
  }))
  default = {}
}

variable "prevent_destroy" {
  description = "Prevent accidental destruction of the database"
  type        = bool
  default     = true
}
