resource "google_pubsub_topic" "notification-registered" {
  name = "notification-registered"
}

resource "google_pubsub_topic_iam_member" "notification-registered-registernotification" {
  project = google_pubsub_topic.notification-registered.project
  topic   = google_pubsub_topic.notification-registered.name
  role    = "roles/pubsub.admin"
  member  = "serviceAccount:${google_service_account.registernotification.email}"
}

resource "google_pubsub_topic_iam_member" "notification-registered-registernotificationaftermath" {
  project = google_pubsub_topic.notification-registered.project
  topic   = google_pubsub_topic.notification-registered.name
  role    = "roles/pubsub.admin"
  member  = "serviceAccount:${google_service_account.registernotificationaftermath.email}"
}
