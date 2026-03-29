package persist

import (
	"log"
	"time"
)

// CascadeLogger provides structured logging for cascade operations.
type CascadeLogger struct {
	operationID string
	startTime   time.Time
}

// NewCascadeLogger creates a new cascade logger.
func NewCascadeLogger() *CascadeLogger {
	return &CascadeLogger{
		startTime: time.Now(),
	}
}

// LogStart logs the start of a cascade operation.
func (l *CascadeLogger) LogStart(event, operationID string) {
	l.operationID = operationID
	log.Printf("[CASCADE] %s: operation_id=%s", event, operationID)
}

// LogInfo logs informational messages.
func (l *CascadeLogger) LogInfo(event, message string) {
	log.Printf("[CASCADE] %s: operation_id=%s message=%s", event, l.operationID, message)
}

// LogError logs error messages.
func (l *CascadeLogger) LogError(event string, err error) {
	log.Printf("[CASCADE] %s: operation_id=%s error=%v", event, l.operationID, err)
}

// LogSuccess logs successful completion.
func (l *CascadeLogger) LogSuccess(event, message string) {
	duration := time.Since(l.startTime)
	log.Printf("[CASCADE] %s: operation_id=%s message=%s duration=%v", event, l.operationID, message, duration)
}

// LogCritical logs critical errors that require manual intervention.
func (l *CascadeLogger) LogCritical(event string, err error) {
	log.Printf("[CASCADE] CRITICAL %s: operation_id=%s error=%v", event, l.operationID, err)
}
