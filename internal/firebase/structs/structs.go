package structs

//Registration DB entity for registration.
type Registration struct {
	Platform                  string `json:"platform"`
	PlatformVersion           string `json:"platformVersion"`
	Manufacturer              string `json:"manufacturer"`
	Model                     string `json:"model"`
	Locale                    string `json:"locale"`
	CreatedAt                 int64  `json:"createdAt"`
	LastNotificationStatus    string `json:"lastNotificationStatus"`
	LastNotificationUpdatedAt int64  `json:"lastNotificationUpdatedAt"`
}

//NotificationCounter DB entity for notification counter.
type NotificationCounter struct {
	NotificationsCount int `json:"notificationsCount"`
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
