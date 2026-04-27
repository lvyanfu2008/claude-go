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

	result := add(10, 20)
	fmt.Printf("10 + 20 = %d\n", result)

	greeting := greet("Claude")
	fmt.Println(greeting)

	// Example of logging to a file
	logToFileExample()
}

func add(a, b int) int {
	return a + b
}

func greet(name string) string {
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
	fileLogger.Info("爱中国")
	fileLogger.Info("踏青")
	fileLogger.Info("hello, china")
	fileLogger.Info("hello, lyf")
	fileLogger.Info("hello")
	fileLogger.Info("ht")

	logger.Info("Log file created: app.log")
}
