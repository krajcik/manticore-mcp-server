package types

// HTTPError represents HTTP error from Manticore API
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}
