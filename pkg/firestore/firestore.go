package firestore

import "time"

// Event is the payload of a Firestore event.
type Event struct {
	OldValue   Value `json:"oldValue"`
	Value      Value `json:"value"`
	UpdateMask struct {
		FieldPaths []string `json:"fieldPaths"`
	} `json:"updateMask"`
}

// Value holds Firestore fields.
type Value struct {
	CreateTime time.Time `json:"createTime"`
	// Fields is the data for this value. The type depends on the format of your
	// database. Log an interface{} value and inspect the result to see a JSON
	// representation of your database fields.
	Fields     Review    `json:"fields"`
	Name       string    `json:"name"`
	UpdateTime time.Time `json:"updateTime"`
}

// Review represents the Firestore schema of a movie review.
type Review struct {
	Author struct {
		Value string `json:"stringValue"`
	} `json:"author"`
	Text struct {
		Value string `json:"stringValue"`
	} `json:"text"`
}
