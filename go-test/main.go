package main

import (
	"fmt"

	"go-test/pkg/mygoword"
)

func main() {
	result := add(10, 20)
	fmt.Printf("10 + 20 = %d\n", result)

	greeting := greet("Claude")
	fmt.Println(greeting)

	logger := mygoword.Default()
	logger.Info("一日千里")
	logger.Info("爱我哈哈")
	logger.Info("爱我ceb")
}

func add(a, b int) int {
	return a + b
}

func greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}
