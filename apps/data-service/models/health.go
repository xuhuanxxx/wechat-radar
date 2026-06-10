package models

// HealthResponse represents the health check response
type HealthResponse struct {
	OK      bool   `json:"ok"`
	Version string `json:"version"`
	Service string `json:"service"`
}

// DoctorResponse represents the doctor check response
type DoctorResponse struct {
	OK     bool           `json:"ok"`
	Checks []DoctorCheck  `json:"checks"`
}

// DoctorCheck represents a single health check
type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}
