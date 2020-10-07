resource "google_pubsub_topic" "notification-registered" {
  name = "notification-registered"
}

resource "google_pubsub_topic" "user-registered" {
  name = "user-registered"
}
