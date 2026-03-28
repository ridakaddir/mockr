package persist

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ridakaddir/mockr/internal/config"
)

// CascadeOperation represents a multi-file mutation operation with transaction semantics.
type CascadeOperation struct {
	ID        string
	Primary   *FileOperation
	Cascades  []*FileOperation
	Backups   map[string][]byte
	Logger    *CascadeLogger
	StartTime time.Time
}

// FileOperation represents a single file operation within a cascade.
type FileOperation struct {
	FilePath       string
	MergeType      string
	FieldPath      string
	Transform      string
	Condition      string
	Data           interface{}
	OriginalExists bool
}

// RequestContext provides context for resolving dynamic paths and conditions.
type RequestContext struct {
	Body        interface{}
	PathParams  map[string]string
	QueryParams map[string]string
	Headers     map[string]string
	// Additional context for proxy integration
	Request      interface{} // *http.Request, but kept as interface{} to avoid import
	BodyBytes    []byte
	ConfigDir    string
	RoutePattern string
}

// ExecuteCascade executes a complete cascade operation with atomic semantics.
func ExecuteCascade(caseConfig config.Case, data interface{}, context RequestContext) error {
	op := &CascadeOperation{
		ID:        generateOperationID(),
		Backups:   make(map[string][]byte),
		Logger:    NewCascadeLogger(),
		StartTime: time.Now(),
	}

	op.Logger.LogStart("cascade_operation_started", op.ID)

	// Phase 1: Validate and prepare all operations
	if err := op.Prepare(caseConfig, data, context); err != nil {
		op.Logger.LogError("cascade_prepare_failed", err)
		return fmt.Errorf("cascade prepare failed: %w", err)
	}

	// Phase 2: Execute all operations with rollback on failure
	if err := op.Execute(); err != nil {
		op.Logger.LogError("cascade_execute_failed", err)

		// Attempt rollback
		if rollbackErr := op.Rollback(); rollbackErr != nil {
			op.Logger.LogCritical("cascade_rollback_failed", rollbackErr)
			return fmt.Errorf("cascade failed and rollback failed: original error: %w, rollback error: %v", err, rollbackErr)
		}

		op.Logger.LogInfo("cascade_rolled_back", "All changes rolled back successfully")
		return fmt.Errorf("cascade operation failed: %w", err)
	}

	duration := time.Since(op.StartTime)
	op.Logger.LogSuccess("cascade_operation_completed", fmt.Sprintf("Operation completed in %v", duration))

	// Notify file watchers after successful completion
	notifyWatchers(op.getAllFilePaths())

	return nil
}

// Prepare validates the cascade configuration and prepares all operations.
func (op *CascadeOperation) Prepare(caseConfig config.Case, data interface{}, context RequestContext) error {
	// Validate cascade configuration
	if caseConfig.Primary == nil {
		return fmt.Errorf("cascade operation requires primary file configuration")
	}

	if len(caseConfig.Cascade) == 0 {
		return fmt.Errorf("cascade operation requires at least one cascade target")
	}

	if len(caseConfig.Cascade) > 10 {
		return fmt.Errorf("too many cascade targets: %d (maximum 10 allowed)", len(caseConfig.Cascade))
	}

	// Prepare primary operation
	primaryPath := resolveFilePathWithContext(caseConfig.Primary.File, context)

	// For primary operation, if field path is specified, use the entire input data
	// The executeUpdate will handle extracting the correct field
	primaryData := data
	op.Primary = &FileOperation{
		FilePath:  primaryPath,
		MergeType: caseConfig.Primary.Merge,
		FieldPath: caseConfig.Primary.Path,
		Data:      primaryData,
	}

	// Create backup for primary file
	if err := op.createBackup(primaryPath); err != nil {
		return fmt.Errorf("failed to create primary backup: %w", err)
	}

	// Prepare cascade operations
	for i, cascadeTarget := range caseConfig.Cascade {
		// Evaluate condition if specified
		if cascadeTarget.Condition != "" {
			shouldExecute, err := evaluateCondition(cascadeTarget.Condition, context)
			if err != nil {
				return fmt.Errorf("failed to evaluate condition for cascade[%d]: %w", i, err)
			}
			if !shouldExecute {
				op.Logger.LogInfo("cascade_target_skipped", fmt.Sprintf("Cascade target %d skipped due to condition", i))
				continue
			}
		}

		// Resolve file pattern to actual files
		targetFiles, err := resolveCascadePattern(cascadeTarget.Pattern, context)
		if err != nil {
			return fmt.Errorf("failed to resolve pattern for cascade[%d]: %w", i, err)
		}

		if len(targetFiles) == 0 {
			return fmt.Errorf("cascade pattern resolved to no files: %s", cascadeTarget.Pattern)
		}

		// Transform data if needed
		targetData := data
		if cascadeTarget.Transform != "" {
			transformed, err := applyTransform(cascadeTarget.Transform, data, context)
			if err != nil {
				return fmt.Errorf("failed to apply transform for cascade[%d]: %w", i, err)
			}
			targetData = transformed
		}

		// Create file operations for each resolved target
		for _, targetFile := range targetFiles {
			// Create backup
			if err := op.createBackup(targetFile); err != nil {
				return fmt.Errorf("failed to create backup for %s: %w", targetFile, err)
			}

			fileOp := &FileOperation{
				FilePath:  targetFile,
				MergeType: cascadeTarget.Merge,
				FieldPath: cascadeTarget.Path,
				Transform: cascadeTarget.Transform,
				Condition: cascadeTarget.Condition,
				Data:      targetData,
			}

			op.Cascades = append(op.Cascades, fileOp)
		}
	}

	op.Logger.LogInfo("cascade_prepared", fmt.Sprintf("Prepared %d cascade operations", len(op.Cascades)))
	return nil
}

// Execute performs all file operations in the cascade.
func (op *CascadeOperation) Execute() error {
	// Execute primary operation
	if err := op.executeFileOperation(op.Primary); err != nil {
		return fmt.Errorf("primary operation failed: %w", err)
	}

	op.Logger.LogInfo("cascade_primary_completed", fmt.Sprintf("Primary file updated: %s", op.Primary.FilePath))

	// Execute cascade operations
	for i, cascadeOp := range op.Cascades {
		if err := op.executeFileOperation(cascadeOp); err != nil {
			return fmt.Errorf("cascade operation %d failed: %w", i, err)
		}
		op.Logger.LogInfo("cascade_target_completed", fmt.Sprintf("Cascade file updated: %s", cascadeOp.FilePath))
	}

	return nil
}

// Rollback restores all files to their original state.
func (op *CascadeOperation) Rollback() error {
	var rollbackErrors []error

	for filePath, originalContent := range op.Backups {
		if originalContent == nil {
			// File didn't exist originally, delete it
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to remove %s: %w", filePath, err))
			}
		} else {
			// Restore original content
			if err := os.WriteFile(filePath, originalContent, 0644); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("failed to restore %s: %w", filePath, err))
			}
		}
	}

	if len(rollbackErrors) > 0 {
		return fmt.Errorf("rollback errors: %v", rollbackErrors)
	}

	return nil
}

// executeFileOperation executes a single file operation.
func (op *CascadeOperation) executeFileOperation(fileOp *FileOperation) error {
	switch fileOp.MergeType {
	case "update":
		return op.executeUpdate(fileOp)
	case "append":
		return op.executeAppend(fileOp)
	case "delete":
		return op.executeDelete(fileOp)
	default:
		return fmt.Errorf("unsupported merge type: %s", fileOp.MergeType)
	}
}

// executeUpdate performs an update operation on a file.
func (op *CascadeOperation) executeUpdate(fileOp *FileOperation) error {
	var updateData map[string]interface{}

	// Convert data to map
	dataMap, ok := fileOp.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("data must be a map for update operations")
	}

	// If field path is specified, handle it appropriately
	if fileOp.FieldPath != "" {
		// Check if this is cascade data (already transformed) or primary data
		if fileOp == op.Primary {
			// For primary operation: extract the field value and create proper nesting
			if fieldValue, exists := dataMap[fileOp.FieldPath]; exists {
				updateData = map[string]interface{}{fileOp.FieldPath: fieldValue}
			} else {
				return fmt.Errorf("field %s not found in primary data", fileOp.FieldPath)
			}
		} else {
			// For cascade operation: data is already transformed, just nest it at the field path
			updateData = createNestedUpdate(fileOp.FieldPath, fileOp.Data)
		}
	} else {
		updateData = dataMap
	}

	op.Logger.LogInfo("executing_update", fmt.Sprintf("Updating file %s with data: %+v", fileOp.FilePath, updateData))

	_, err := Update(fileOp.FilePath, updateData)
	return err
}

// executeAppend performs an append operation to a directory.
func (op *CascadeOperation) executeAppend(fileOp *FileOperation) error {
	// Convert data to map for AppendToDir function
	dataMap, ok := fileOp.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("data must be a map for append operations")
	}

	// Extract key if available
	key := ""
	if keyField, exists := dataMap["id"]; exists {
		if keyStr, ok := keyField.(string); ok {
			key = keyStr
		}
	}

	_, _, err := AppendToDir(fileOp.FilePath, key, dataMap)
	return err
}

// executeDelete performs a delete operation on a file.
func (op *CascadeOperation) executeDelete(fileOp *FileOperation) error {
	return DeleteFile(fileOp.FilePath)
}

// createNestedUpdate creates a nested map structure for field path updates.
func createNestedUpdate(fieldPath string, data interface{}) map[string]interface{} {
	parts := strings.Split(fieldPath, ".")
	if len(parts) == 1 {
		// Single level field path, create wrapper
		return map[string]interface{}{parts[0]: data}
	}

	result := make(map[string]interface{})
	current := result

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last level, add the actual data
			current[part] = data
		} else {
			// Create intermediate level
			next := make(map[string]interface{})
			current[part] = next
			current = next
		}
	}

	return result
}

// createBackup creates a backup of a file if it exists.
func (op *CascadeOperation) createBackup(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, store nil to indicate this
			op.Backups[filePath] = nil
			return nil
		}
		return err
	}

	op.Backups[filePath] = content
	return nil
}

// getAllFilePaths returns all file paths involved in the cascade operation.
func (op *CascadeOperation) getAllFilePaths() []string {
	paths := []string{op.Primary.FilePath}
	for _, cascade := range op.Cascades {
		paths = append(paths, cascade.FilePath)
	}
	return paths
}

// generateOperationID generates a unique operation ID.
func generateOperationID() string {
	return "cascade-op-" + uuid.New().String()[:8]
}

// resolveFilePathWithContext resolves file paths using proxy logic when available
func resolveFilePathWithContext(pattern string, context RequestContext) string {
	// If we have proxy context, use the full resolution logic
	if context.Request != nil && context.ConfigDir != "" {
		// This would need proper proxy integration, for now use simple resolution
		return resolveFilePath(pattern, context)
	}
	return resolveFilePath(pattern, context)
}

// Helper function to resolve file paths with context placeholders
func resolveFilePath(pattern string, context RequestContext) string {
	resolved := pattern

	// Replace path parameters
	for key, value := range context.PathParams {
		placeholder := fmt.Sprintf("{path.%s}", key)
		resolved = strings.ReplaceAll(resolved, placeholder, value)
	}

	// Replace query parameters
	for key, value := range context.QueryParams {
		placeholder := fmt.Sprintf("{query.%s}", key)
		resolved = strings.ReplaceAll(resolved, placeholder, value)
	}

	// Make path absolute with config directory if available
	if context.ConfigDir != "" {
		if !strings.HasPrefix(resolved, "/") {
			resolved = filepath.Join(context.ConfigDir, resolved)
		}
	}

	return resolved
}
