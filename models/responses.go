// path: models/responses.go
package models

// LocateRequest is the request body for POST /api/locate.
type LocateRequest struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// LocateResponse is the response body for POST /api/locate.
type LocateResponse struct {
	Label     string `json:"label"`
	AreaLabel string `json:"area_label"`
}

// ReportCreatePayload is the JSON body for POST /api/reports (JSON branch).
// (Multipart branch reads fields from form directly.)
type ReportCreatePayload struct {
	Category       string  `json:"category"`
	Note           string  `json:"note"`
	AreaLabel      string  `json:"area_label"`
	Lat            float64 `json:"lat"`
	Lng            float64 `json:"lng"`
	AccuracyM      *int    `json:"accuracy_m"`
	PrivacyRadiusM *int    `json:"privacy_radius_m"`
	Anonymous      bool    `json:"anonymous"`
}
type CreateReportResp struct {
	OK    bool   `json:"ok"`
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}
