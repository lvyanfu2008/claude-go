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

	// Print china lives long message
	logger.Info("china lives long")

	// Print hello lyf message
	logger.Info("hello lyf")

	// Add hello and lvyanfu logs
	logger.Info("hello")
	logger.Info("lvyanfu")

	// Add hello,cc log
	logger.Info("hello,cc")

	// Add hello,lv.yanfu log
	logger.Info("hello,lv.yanfu")

	// Add hello,yr log
	logger.Info("hello,jyr")

	// Add hello日志
	logger.Info("hello日志")

	// Add hello,lll日志
	logger.Info("hello,lll")

	// Add hello,LYF日志
	logger.Info("hello,LYF")

	// Add 爱我中华日志
	logger.Info("爱我中华")
	
	// Add 爱我国家日志
	logger.Info("爱我国家")

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
	fileLogger.Info("hello")
	fileLogger.Info("lvyanfu")

	logger.Info("Log file created: app.log")
}
