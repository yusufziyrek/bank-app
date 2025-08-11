package dto

type ErrorResponse struct {
	Code             string            `json:"code"`
	Message          string            `json:"message"`
	Details          string            `json:"details,omitempty"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
}

type ValidationError struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}
