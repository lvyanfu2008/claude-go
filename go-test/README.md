# Go Test Project

这是一个用于 Go 语言测试和学习的项目。

## 项目结构

```
go-test/
├── main.go          # 主程序文件
├── main_test.go     # 测试文件
├── go.mod           # Go 模块文件
└── README.md        # 项目说明文档
```

## 功能

1. **基本数学运算**：包含加法函数 `add()`
2. **字符串处理**：包含问候函数 `greet()`
3. **单元测试**：包含完整的测试用例

## 使用方法

### 运行程序
```bash
cd go-test
go run main.go
```

### 运行测试
```bash
go test
```

### 运行测试并查看详细信息
```bash
go test -v
```

### 运行特定测试
```bash
go test -run TestAdd
go test -run TestGreet
```

## 示例输出

运行 `go run main.go` 会输出：
```
Go Test Project
This is a test project for Go development
10 + 20 = 30
Hello, Claude!
```

## 测试覆盖率

查看测试覆盖率：
```bash
go test -cover
```

生成覆盖率报告：
```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```