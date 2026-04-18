// Package mygoword provides a simple logging utility for Go applications.
//
// Example usage:
//
//	logger := mygoword.Default()
//	logger.Info("Application started")
//	logger.Debug("Processing item %d", 42)
//	logger.Error("Failed to connect: %v", err)
//
// You can also create a logger with custom configuration:
//
//	logger := mygoword.New(mysgoword.DEBUG, os.Stderr)
//	logger.SetLevel(mysgoword.WARN)
package mygoword

import "fmt"

// Example demonstrates basic usage of the logger.
func Example() {
	// Create a default logger (INFO level, stdout output)
	logger := Default()
	
	// Log messages at different levels
	logger.Debug("This is a debug message") // Won't be logged with INFO level
	logger.Info("Application started")
	logger.Warn("Disk space is running low")
	logger.Error("Failed to read config file")
	
	// Change log level to DEBUG
	logger.SetLevel(DEBUG)
	logger.Debug("Now debug messages will be shown")
	
	// Log with formatting
	count := 5
	logger.Info("Processed %d items", count)
	
	// Output:
	// [timestamp] INFO: Application started
	// [timestamp] WARN: Disk space is running low
	// [timestamp] ERROR: Failed to read config file
	// [timestamp] DEBUG: Now debug messages will be shown
	// [timestamp] INFO: Processed 5 items
}

// ExampleWithFields demonstrates using the logger with context fields.
func ExampleWithFields() {
	logger := Default()
	
	// Create a logger with context fields
	contextLogger := logger.WithFields(map[string]interface{}{
		"service": "auth",
		"version": "1.0.0",
	})
	
	contextLogger.Info("User login attempt")
	contextLogger.Warn("Invalid credentials")
	
	// Output would include context fields in a real implementation
	fmt.Println("Note: WithFields is a simplified implementation")
}