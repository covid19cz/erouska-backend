resource "google_pubsub_topic" "notification-registered" {
  name = "notification-registered"
}

resource "google_pubsub_topic" "user-registered" {
  name = "user-registered"
}

resource "google_pubsub_topic" "efgs-import-keys" {
  name = "efgs-import-keys"
}

resource "google_pubsub_topic" "efgs-postponed-yesterdays-downloading" {
  name = "efgs-postponed-yesterdays-downloading"
}
