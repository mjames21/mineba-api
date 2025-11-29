// path: models/report.go
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Report struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Category       string             `bson:"category" json:"category"`
	Note           string             `bson:"note" json:"note"`
	AreaLabel      string             `bson:"area_label" json:"area_label"`
	Lat            float64            `bson:"lat" json:"lat"`
	Lng            float64            `bson:"lng" json:"lng"`
	AccuracyM      *int               `bson:"accuracy_m,omitempty" json:"accuracy_m,omitempty"`
	PrivacyRadiusM *int               `bson:"privacy_radius_m,omitempty" json:"privacy_radius_m,omitempty"`
	Anonymous      bool               `bson:"anonymous" json:"anonymous"`

	// Media (voice is optional, we store all media in PhotoURLs to keep it simple)
	VoiceURL  string   `bson:"voice_url,omitempty" json:"voice_url,omitempty"`
	PhotoURLs []string `bson:"photo_urls,omitempty" json:"photo_urls,omitempty"`

	// Address analytics (optional)
	Adrehs    string `bson:"adrehs,omitempty" json:"adrehs,omitempty"`
	District  string `bson:"district,omitempty" json:"district,omitempty"`
	Chiefdom  string `bson:"chiefdom,omitempty" json:"chiefdom,omitempty"`
	Region    string `bson:"region,omitempty" json:"region,omitempty"`
	Section   string `bson:"section,omitempty" json:"section,omitempty"`
	GeoMethod string `bson:"geo_method,omitempty" json:"geo_method,omitempty"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
