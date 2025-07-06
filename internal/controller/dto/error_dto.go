package dto

type ErrorResponse struct {
	Message          string            `json:"message"`
	Code             string            `json:"code,omitempty"`
	Details          string            `json:"details,omitempty"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
}

type ValidationError struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}
