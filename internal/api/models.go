package api

// APIResponse is the standardized response structure for ALL HTTP endpoints
// This structure MUST be used by every endpoint in the application
type APIResponse struct {
	OK      bool        `json:"ok"`                // true if successful, false if error
	Code    int         `json:"code"`              // Response code: positive (success), negative (error)
	Message string      `json:"message"`           // Human-readable message
	Result  interface{} `json:"result,omitempty"`  // Response data (optional)
	Meta    interface{} `json:"meta,omitempty"`    // Metadata (pagination, etc.)
}

// NewSuccessResponse creates a successful API response
func NewSuccessResponse(code int, message string, result interface{}) *APIResponse {
	return &APIResponse{
		OK:      true,
		Code:    code,
		Message: message,
		Result:  result,
	}
}

// NewSuccessResponseWithMeta creates a successful API response with metadata
func NewSuccessResponseWithMeta(code int, message string, result interface{}, meta interface{}) *APIResponse {
	return &APIResponse{
		OK:      true,
		Code:    code,
		Message: message,
		Result:  result,
		Meta:    meta,
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(code int, message string) *APIResponse {
	return &APIResponse{
		OK:      false,
		Code:    code,
		Message: message,
		Result:  nil,
	}
}

// NewErrorResponseWithMeta creates an error API response with metadata
func NewErrorResponseWithMeta(code int, message string, meta interface{}) *APIResponse {
	return &APIResponse{
		OK:      false,
		Code:    code,
		Message: message,
		Result:  nil,
		Meta:    meta,
	}
}

// Response codes (application-specific)
// Positive codes = Success operations
// Negative codes = Error operations
const (
	// Success codes (1-999)
	CodeSuccess           = 1   // Generic success
	CodeDataRetrieved     = 10  // Data retrieved successfully
	CodeDataCreated       = 11  // Data created successfully
	CodeDataUpdated       = 12  // Data updated successfully
	CodeDataDeleted       = 13  // Data deleted successfully
	CodeSyncSuccess       = 20  // Sync operation successful
	CodeServiceStarted    = 30  // Service started successfully
	CodeServiceStopped    = 31  // Service stopped successfully
	CodeServiceRestarted  = 32  // Service restarted successfully
	CodeConfigUpdated     = 40  // Configuration updated

	// Error codes (-1 to -999)
	CodeErrorGeneric      = -1   // Generic error
	CodeErrorBadRequest   = -10  // Invalid request parameters
	CodeErrorUnauthorized = -11  // Unauthorized access
	CodeErrorForbidden    = -12  // Forbidden operation
	CodeErrorNotFound     = -13  // Resource not found
	CodeErrorDatabase     = -20  // Database error
	CodeErrorEncryption   = -21  // Encryption/Decryption error
	CodeErrorConfig       = -22  // Configuration error
	CodeErrorSync         = -30  // Synchronization error
	CodeErrorOffline      = -31  // Service offline too long
	CodeErrorService      = -40  // Windows service error
	CodeErrorInternal     = -99  // Internal server error
)

// Common response messages
const (
	MessageSuccess              = "Success"
	MessageCreated              = "Resource created successfully"
	MessageUpdated              = "Resource updated successfully"
	MessageDeleted              = "Resource deleted successfully"
	MessageBadRequest           = "Invalid request parameters"
	MessageUnauthorized         = "Unauthorized"
	MessageForbidden            = "Forbidden"
	MessageNotFound             = "Resource not found"
	MessageInternalError        = "Internal server error"
	MessageServiceUnavailable   = "Service unavailable"
	MessageOfflineTooLong       = "Service offline for more than 24 hours"
)

// ServiceStatus represents the current service status
type ServiceStatus struct {
	Status          string `json:"status"`            // "running", "stopped", "offline"
	LastSyncTime    string `json:"last_sync_time"`    // ISO 8601 timestamp
	OfflineHours    int    `json:"offline_hours"`     // Hours since last successful sync
	IsHealthy       bool   `json:"is_healthy"`        // Overall health status
	WindowsService  string `json:"windows_service"`   // "running", "stopped"
}

// HealthCheck represents the health check response
type HealthCheck struct {
	Healthy       bool   `json:"healthy"`
	Version       string `json:"version"`
	Timestamp     string `json:"timestamp"`
	DatabaseOK    bool   `json:"database_ok"`
	ConfigOK      bool   `json:"config_ok"`
}
