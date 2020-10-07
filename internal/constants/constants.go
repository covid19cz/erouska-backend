package constants

//ProjectID GCP project ID
const ProjectID = "erouska-key-server-dev"

//FirebaseURL URL to our firebase DB.
const FirebaseURL = "https://" + ProjectID + ".firebaseio.com/"

//CollectionRegistrations Name of the collection.
const CollectionRegistrations = "registrations_v2"

//CollectionDailyNotificationAttemptsEhrid Name of the collection.
const CollectionDailyNotificationAttemptsEhrid = "dailyNotificationAttemptsEhrid"

//CollectionNotificationCounters Name of the collection.
const CollectionNotificationCounters = "notificationCounters"

//CollectionCovidDataTotal Name of the collection.
const CollectionCovidDataTotal = "covidDataTotal"

//TopicRegisterNotification Name of the topic.
const TopicRegisterNotification = "notification-registered"

//TopicRegisterUser Name of the topic.
const TopicRegisterUser = "user-registered"

//DbUserCountersPrefix Prefix of user counters data in Realtime DB.
const DbUserCountersPrefix = "userCounters/"
