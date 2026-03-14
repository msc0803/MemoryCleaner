package main

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatus         = kernel32.NewProc("GlobalMemoryStatusEx")
	setProcessWorkingSetSize   = kernel32.NewProc("SetProcessWorkingSetSize")
	getCurrentProcess          = kernel32.NewProc("GetCurrentProcess")
	psapi                      = syscall.NewLazyDLL("psapi.dll")
)

type MEMORYSTATUSEX struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func main() {
	a := app.New()
	w := a.NewWindow("Memory Cleaner - 内存清理工具")

	totalLabel := widget.NewLabel("总物理内存: 计算中...")
	availLabel := widget.NewLabel("可用内存: 计算中...")
	usedLabel := widget.NewLabel("已用内存: 计算中...")
	usageLabel := widget.NewLabel("使用率: 计算中...")
	statusLabel := widget.NewLabel("就绪")

	cleanBtn := widget.NewButton("清理物理内存", func() {
		statusLabel.SetText("正在清理物理内存...")
		freed := cleanPhysicalMemory()
		statusLabel.SetText(fmt.Sprintf("清理完成! 释放约 %s", formatBytes(freed)))
		updateLabels(totalLabel, availLabel, usedLabel, usageLabel)
	})

	workingSetBtn := widget.NewButton("清理工作集", func() {
		statusLabel.SetText("正在清理工作集...")
		cleanWorkingSet()
		statusLabel.SetText("工作集清理完成!")
		updateLabels(totalLabel, availLabel, usedLabel, usageLabel)
	})

	autoCleanBtn := widget.NewButton("启动自动清理 (每30秒)", nil)
	var autoCleanRunning bool
	var stopChan chan struct{}
	
	autoCleanBtn.OnTapped = func() {
		if !autoCleanRunning {
			autoCleanRunning = true
			autoCleanBtn.SetText("停止自动清理")
			statusLabel.SetText("自动清理已启动 (每30秒)")
			stopChan = make(chan struct{})
			go func() {
				ticker := time.NewTicker(30 * time.Second)
				for {
					select {
					case <-ticker.C:
						cleanPhysicalMemory()
					case <-stopChan:
						ticker.Stop()
						return
					}
				}
			}()
		} else {
			autoCleanRunning = false
			autoCleanBtn.SetText("启动自动清理 (每30秒)")
			statusLabel.SetText("自动清理已停止")
			close(stopChan)
		}
	}

	content := container.NewVBox(
		widget.NewLabel("=== 内存状态 ==="),
		totalLabel,
		availLabel,
		usedLabel,
		usageLabel,
		widget.NewSeparator(),
		widget.NewLabel("=== 操作 ==="),
		cleanBtn,
		workingSetBtn,
		autoCleanBtn,
		widget.NewSeparator(),
		statusLabel,
	)

	// 更新状态
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for range ticker.C {
			updateLabels(totalLabel, availLabel, usedLabel, usageLabel)
		}
	}()

	updateLabels(totalLabel, availLabel, usedLabel, usageLabel)
	w.SetContent(content)
	w.Resize(fyne.NewSize(350, 400))
	w.ShowAndRun()
}

func updateLabels(totalLabel, availLabel, usedLabel, usageLabel *widget.Label) {
	memStatus := getMemoryStatus()
	used := memStatus.ullTotalPhys - memStatus.ullAvailPhys
	
	totalLabel.SetText(fmt.Sprintf("总物理内存: %s", formatBytes(memStatus.ullTotalPhys)))
	availLabel.SetText(fmt.Sprintf("可用内存: %s", formatBytes(memStatus.ullAvailPhys)))
	usedLabel.SetText(fmt.Sprintf("已用内存: %s", formatBytes(used)))
	usageLabel.SetText(fmt.Sprintf("使用率: %d%%", memStatus.dwMemoryLoad))
}

func cleanPhysicalMemory() uint64 {
	before := getMemoryStatus()
	
	handle, _, _ := getCurrentProcess.Call()
	setProcessWorkingSetSize.Call(handle, ^uintptr(0), ^uintptr(0))
	
	emptyWorkingSet := psapi.NewProc("EmptyWorkingSet")
	emptyWorkingSet.Call(handle)
	
	runtime.GC()
	
	after := getMemoryStatus()
	freed := int64(before.ullTotalPhys - before.ullAvailPhys) - int64(after.ullTotalPhys - after.ullAvailPhys)
	if freed < 0 {
		return 0
	}
	return uint64(freed)
}

func cleanWorkingSet() {
	handle, _, _ := getCurrentProcess.Call()
	setProcessWorkingSetSize.Call(handle, ^uintptr(0), ^uintptr(0))
}

func getMemoryStatus() MEMORYSTATUSEX {
	var memStatus MEMORYSTATUSEX
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))
	globalMemoryStatus.Call(uintptr(unsafe.Pointer(&memStatus)))
	return memStatus
}

func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
