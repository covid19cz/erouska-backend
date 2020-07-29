package structs

//Registration DB entity for registration.
type Registration struct {
	Platform                  string `json:"platform"`
	PlatformVersion           string `json:"platformVersion"`
	Manufacturer              string `json:"manufacturer"`
	Model                     string `json:"model"`
	Locale                    string `json:"locale"`
	CreatedAt                 string `json:"createdAt"`
	LastNotificationStatus    string `json:"lastNotificationStatus"`
	LastNotificationUpdatedAt string `json:"lastNotificationUpdatedAt"`
}
