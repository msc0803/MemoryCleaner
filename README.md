# Memory Cleaner - 内存清理工具

一个轻量级的 Windows 内存清理工具，用 Go 语言编写。

## 功能

- 实时显示内存状态（总内存、可用内存、已用内存、使用率）
- 清理物理内存
- 清理工作集
- 自动清理（每30秒）

## 编译

### 在 Windows 上编译

```bash
go mod tidy
go build -ldflags="-H windowsgui -s -w" -o MemoryCleaner.exe .
```

### 在 macOS/Linux 上交叉编译到 Windows

需要安装 MinGW-w64 交叉编译器：

```bash
# macOS
brew install mingw-w64

# 编译
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
  go build -ldflags="-H windowsgui -s -w" -o MemoryCleaner.exe .
```

## 使用

直接双击 `MemoryCleaner.exe` 运行。

## 注意

- 此工具通过 Windows API 调用 `SetProcessWorkingSetSize` 和 `EmptyWorkingSet` 来释放内存
- 这是安全的标准做法，不会造成数据丢失
- 编译后的 exe 文件约 2-3MB，无任何外部依赖
