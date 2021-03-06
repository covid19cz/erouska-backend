package structs

//Registration DB entity for registration.
type Registration struct {
	Platform                  string `json:"platform"`
	PlatformVersion           string `json:"platformVersion"`
	Manufacturer              string `json:"manufacturer"`
	Model                     string `json:"model"`
	Locale                    string `json:"locale"`
	PushRegistrationToken     string `json:"pushRegistrationToken"`
	CreatedAt                 int64  `json:"createdAt"`
	LastNotificationStatus    string `json:"lastNotificationStatus"`
	LastNotificationUpdatedAt int64  `json:"lastNotificationUpdatedAt"`
}

//RegistrationV1 DB entity for registration V1 - the legacy one.
type RegistrationV1 struct {
	FUID                  string `firestore:"fuid" json:"fuid"`
	Platform              string `firestore:"platform" json:"platform"`
	PlatformVersion       string `firestore:"platformVersion" json:"platformVersion"`
	Manufacturer          string `firestore:"manufacturer" json:"manufacturer"`
	Model                 string `firestore:"model" json:"model"`
	Locale                string `firestore:"locale" json:"locale"`
	PushRegistrationToken string `firestore:"pushRegistrationToken" json:"pushRegistrationToken"`
	CreatedAt             int64  `firestore:"createdAt" json:"createdAt"`
}

//NotificationCounter DB entity for notification counter.
type NotificationCounter struct {
	NotificationsCount int `json:"notificationsCount"`
}

//UserCounter DB entity for users counter.
type UserCounter struct {
	UsersCount int `json:"usersCount"`
}

//PublisherCounter DB entity for publishers counter.
type PublisherCounter struct {
	PublishersCount int `json:"publishersCount"`
	KeysCount       int `json:"keysCount"`
}

//EfgsCounter DB entity for EFGS counters.
type EfgsCounter struct {
	KeysUploaded   int `json:"keysUploaded"`
	KeysDownloaded int `json:"keysDownloaded"`
	KeysImportedCZ int `json:"keysImportedCZ"`
	Publishers     int `json:"publishers"`
}

//VerificationCodeMetadata DB entity for verification code metadata.
type VerificationCodeMetadata struct {
	VsMetadata map[string]interface{} `json:"vsMetadata"`
	IssuedAt   int64                  `json:"issuedAt"`
}

// IntegerValue represents integer (as string) in firestore events
type IntegerValue struct {
	IntegerValue string `json:"integerValue"`
}

// StringValue represents strings in firestore events
type StringValue struct {
	StringValue string `json:"stringValue"`
}

//MetricsData Data of metrics.
type MetricsData struct {
	Modified                    int64  `json:"modified"`
	Date                        string `json:"date"`
	ActivationsYesterday        int32  `json:"activations_yesterday"`
	ActivationsTotal            int32  `json:"activations_total"`
	KeyPublishersYesterday      int32  `json:"key_publishers_yesterday"`
	KeyPublishersTotal          int32  `json:"key_publishers_total"`
	NotificationsYesterday      int32  `json:"notifications_yesterday"`
	NotificationsTotal          int32  `json:"notifications_total"`
	EfgsKeysUploadedTotal       int32  `json:"efgs_keys_uploaded_total"`
	EfgsKeysUploadedYesterday   int32  `json:"efgs_keys_uploaded_yesterday"`
	EfgsKeysDownloadedTotal     int32  `json:"efgs_keys_downloaded_total"`
	EfgsKeysDownloadedYesterday int32  `json:"efgs_keys_downloaded_yesterday"`
	EfgsPublishersTotal         int32  `json:"efgs_publishers_total"`
	EfgsPublishersYesterday     int32  `json:"efgs_publishers_yesterday"`
	EfgsImportedCzTotal         int32  `json:"efgs_imported_cz_total"`
	EfgsImportedCzYesterday     int32  `json:"efgs_imported_cz_yesterday"`
}
