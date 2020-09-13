package v1

import (
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
)

/*
This files contains request/response structs for the PublishKeys endpoint. The structs have to be changed in
backward-compatible way and when it's not possible, copied to `v2` and changed there.

The base of these structs is copied from https://github.com/google/exposure-notifications-server/blob/main/pkg/api/v1/exposure_types.go
*/

// PublishKeysRequestDevice represents the body of the PublishInfectedIds API call. It's received from device.
//
// VisitedCountries: list (possibly empty) where the device has travelled to.
//
// ReportType: type of report - is it self-report, confirmed diagnose, ...?
//
// ConsentToFederation: whether user of the device has allowed to share his keys to other countries.
type PublishKeysRequestDevice struct {
	keyserverapi.Publish // embedded struct

	VisitedCountries    []string   `json:"visitedCountries"`
	ReportType          ReportType `json:"reportType"`
	ConsentToFederation bool       `json:"consentToFederation"`
}

// PublishKeysResponseDevice is sent back to the client on a publish request.
type PublishKeysResponseDevice keyserverapi.PublishResponse

// DeviceExposureKey is the 16 byte key, the start time of the key and the duration of the key. A duration of 0 means 24 hours.
type DeviceExposureKey keyserverapi.ExposureKey

// PublishKeysRequestServer represents the body of the PublishInfectedIds API call. It's sent to the key server.
type PublishKeysRequestServer keyserverapi.Publish

// PublishKeysResponseServer is got back from the key server on a publish request.
type PublishKeysResponseServer keyserverapi.PublishResponse

//ReportType means type of the keys report - is it self-report, confirmed diagnose, ...?
type ReportType string

// Possible values of ReportType.
const (
	Unknown                    = "Unknown"
	ConfirmedTest              = "ConfirmedTest"
	ConfirmedClinicalDiagnosis = "ConfirmedClinicalDiagnosis"
	SelfReport                 = "SelfReport"
	Recursive                  = "Recursive"
	Revoked                    = "Revoked"
)
