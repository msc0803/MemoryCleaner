package main

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatus = kernel32.NewProc("GlobalMemoryStatusEx")
	setProcessWorkingSetSize = kernel32.NewProc("SetProcessWorkingSetSize")
	getCurrentProcess  = kernel32.NewProc("GetCurrentProcess")
	emptyWorkingSet    = kernel32.NewProc("EmptyWorkingSet")
	psapi              = syscall.NewLazyDLL("psapi.dll")
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

type MainWindow struct {
	*walk.MainWindow
	totalMemory     *walk.Label
	availableMemory *walk.Label
	usedMemory      *walk.Label
	usagePercent    *walk.ProgressBar
	statusLabel     *walk.Label
}

func main() {
	mw := &MainWindow{}
	
	err := MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "Memory Cleaner - 内存清理工具",
		MinSize:  Size{Width: 400, Height: 350},
		Size:     Size{Width: 450, Height: 400},
		Layout:   VBox{},
		Children: []Widget{
			GroupBox{
				Title:  "内存状态",
				Layout: VBox{},
				Children: []Widget{
					Label{Text: "总物理内存:"},
					Label{AssignTo: &mw.totalMemory, Text: "计算中..."},
					Label{Text: "可用内存:"},
					Label{AssignTo: &mw.availableMemory, Text: "计算中..."},
					Label{Text: "已用内存:"},
					Label{AssignTo: &mw.usedMemory, Text: "计算中..."},
					Label{Text: "使用率:"},
					ProgressBar{
						AssignTo: &mw.usagePercent,
						MinValue: 0,
						MaxValue: 100,
					},
				},
			},
			GroupBox{
				Title:  "操作",
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text:      "清理物理内存",
						OnClicked: mw.cleanPhysicalMemory,
					},
					PushButton{
						Text:      "清理工作集",
						OnClicked: mw.cleanWorkingSet,
					},
				},
			},
			PushButton{
				Text:      "自动清理 (每30秒)",
				OnClicked: mw.toggleAutoClean,
			},
			Label{AssignTo: &mw.statusLabel, Text: "就绪"},
		},
	}.Create()
	
	if err != nil {
		fmt.Println("创建窗口失败:", err)
		return
	}

	mw.updateMemoryStatus()
	go mw.autoUpdateStatus()
	
	mw.Run()
}

func (mw *MainWindow) updateMemoryStatus() {
	memStatus := getMemoryStatus()
	
	mw.totalMemory.SetText(formatBytes(memStatus.ullTotalPhys))
	mw.availableMemory.SetText(formatBytes(memStatus.ullAvailPhys))
	used := memStatus.ullTotalPhys - memStatus.ullAvailPhys
	mw.usedMemory.SetText(formatBytes(used))
	mw.usagePercent.SetValue(int(memStatus.dwMemoryLoad))
}

func (mw *MainWindow) autoUpdateStatus() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		mw.updateMemoryStatus()
	}
}

func (mw *MainWindow) cleanPhysicalMemory() {
	mw.statusLabel.SetText("正在清理物理内存...")
	
	before := getMemoryStatus()
	
	// 方法1: 通过调整工作集大小来释放内存
	handle, _, _ := getCurrentProcess.Call()
	setProcessWorkingSetSize.Call(handle, ^uintptr(0), ^uintptr(0))
	
	// 方法2: 清空工作集
	psapi := syscall.NewLazyDLL("psapi.dll")
	emptyWorkingSet := psapi.NewProc("EmptyWorkingSet")
	emptyWorkingSet.Call(handle)
	
	// 触发GC
	runtime.GC()
	
	after := getMemoryStatus()
	freed := before.ullTotalPhys - before.ullAvailPhys - (after.ullTotalPhys - after.ullAvailPhys)
	if freed < 0 {
		freed = 0
	}
	
	mw.statusLabel.SetText(fmt.Sprintf("清理完成! 释放约 %s", formatBytes(uint64(freed))))
	mw.updateMemoryStatus()
}

func (mw *MainWindow) cleanWorkingSet() {
	mw.statusLabel.SetText("正在清理工作集...")
	
	handle, _, _ := getCurrentProcess.Call()
	setProcessWorkingSetSize.Call(handle, ^uintptr(0), ^uintptr(0))
	
	mw.statusLabel.SetText("工作集清理完成!")
	mw.updateMemoryStatus()
}

var autoCleanRunning bool

func (mw *MainWindow) toggleAutoClean() {
	autoCleanRunning = !autoCleanRunning
	if autoCleanRunning {
		mw.statusLabel.SetText("自动清理已启动 (每30秒)")
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			for range ticker.C {
				if !autoCleanRunning {
					return
				}
				mw.cleanPhysicalMemory()
			}
		}()
	} else {
		mw.statusLabel.SetText("自动清理已停止")
	}
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
