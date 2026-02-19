resource "sequin_sink_consumer" "kafka" {
  name     = "orders-to-kafka"
  database = sequin_database.main.id

  tables = [
    { name = "public.orders", group_column_names = ["id"] },
    { name = "public.order_items" }
  ]

  actions = ["insert", "update"]

  destination = {
    type           = "kafka"
    hosts          = "broker1:9092,broker2:9092"
    topic          = "database.orders"
    tls            = true
    username       = "kafka-user"
    password       = var.kafka_password
    sasl_mechanism = "SCRAM-SHA-256"
  }

  batch_size       = 100
  message_grouping = true
}

resource "sequin_sink_consumer" "sqs" {
  name     = "events-to-sqs"
  database = sequin_database.main.id

  tables  = [{ name = "public.events" }]
  actions = ["insert", "update"]

  destination = {
    type              = "sqs"
    queue_url         = "https://sqs.us-east-1.amazonaws.com/123456789012/events"
    region            = "us-east-1"
    access_key_id     = var.aws_access_key
    secret_access_key = var.aws_secret_key
    is_fifo           = false
  }
}

resource "sequin_sink_consumer" "kinesis" {
  name     = "events-to-kinesis"
  database = sequin_database.main.id

  tables  = [{ name = "public.events", group_column_names = ["tenant_id"] }]
  actions = ["insert", "update", "delete"]

  destination = {
    type              = "kinesis"
    stream_arn        = "arn:aws:kinesis:us-east-1:123456789012:stream/events"
    access_key_id     = var.aws_access_key
    secret_access_key = var.aws_secret_key
  }

  message_grouping = true
}

resource "sequin_sink_consumer" "webhook" {
  name     = "notifications-webhook"
  database = sequin_database.main.id

  tables  = [{ name = "public.notifications" }]
  actions = ["insert"]

  destination = {
    type               = "webhook"
    http_endpoint      = "https://api.example.com"
    http_endpoint_path = "/webhook/notifications"
    batch              = true
  }

  filter    = "my-filter-function"
  transform = "my-transform-function"
}
