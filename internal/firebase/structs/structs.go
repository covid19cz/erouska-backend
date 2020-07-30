package structs

type Registration struct {
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platformVersion"`
	Manufacturer    string `json:"manufacturer"`
	Model           string `json:"model"`
	Locale          string `json:"locale"`
}
