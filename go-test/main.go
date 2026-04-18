package main

import (
	"fmt"
	"go-test/pkg/mygoword"
	"os"
)

var logger = mygoword.Default()

func main() {
	// Set log level to DEBUG for more verbose output
	logger.SetLevel(mygoword.DEBUG)
	
	logger.Info("Go Test Project starting...")
	logger.Debug("This is a debug message")
	
	// Example function calls with logging
	logger.Info("Performing addition operation")
	result := add(10, 20)
	logger.Debug("add(10, 20) = %d", result)
	fmt.Printf("10 + 20 = %d\n", result)
	
	logger.Info("Generating greeting")
	greeting := greet("Claude")
	logger.Debug("greet(\"Claude\") = %s", greeting)
	fmt.Println(greeting)
	
	// Demonstrate different log levels
	logger.Warn("This is a warning message")
	logger.Error("This is an error message (simulated)")
	
	// Example of logging to a file
	logToFileExample()
	
	logger.Info("Application completed successfully")
}

func add(a, b int) int {
	logger.Debug("Adding %d and %d", a, b)
	return a + b
}

func greet(name string) string {
	logger.Debug("Greeting %s", name)
	return fmt.Sprintf("Hello, %s!", name)
}

func logToFileExample() {
	// Create a log file
	file, err := os.Create("app.log")
	if err != nil {
		logger.Error("Failed to create log file: %v", err)
		return
	}
	defer file.Close()
	
	// Create a new logger that writes to the file
	fileLogger := mygoword.New(mygoword.INFO, file)
	fileLogger.Info("This log message goes to the file")
	fileLogger.Info("Application log entry")
	
	logger.Info("Log file created: app.log")
}