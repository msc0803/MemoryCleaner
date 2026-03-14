package main

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	user32                   = syscall.NewLazyDLL("user32.dll")
	gdi32                    = syscall.NewLazyDLL("gdi32.dll")
	psapi                    = syscall.NewLazyDLL("psapi.dll")
	
	globalMemoryStatus       = kernel32.NewProc("GlobalMemoryStatusEx")
	setProcessWorkingSetSize = kernel32.NewProc("SetProcessWorkingSetSize")
	getCurrentProcess        = kernel32.NewProc("GetCurrentProcess")
	getModuleHandle          = kernel32.NewProc("GetModuleHandleW")
	
	createWindowEx           = user32.NewProc("CreateWindowExW")
	defWindowProc            = user32.NewProc("DefWindowProcW")
	registerClassEx          = user32.NewProc("RegisterClassExW")
	getMessage               = user32.NewProc("GetMessageW")
	translateMessage         = user32.NewProc("TranslateMessage")
	dispatchMessage          = user32.NewProc("DispatchMessageW")
	postQuitMessage          = user32.NewProc("PostQuitMessage")
	setWindowText            = user32.NewProc("SetWindowTextW")
	getClientRect            = user32.NewProc("GetClientRect")
	invalidateRect           = user32.NewProc("InvalidateRect")
	beginPaint               = user32.NewProc("BeginPaint")
	endPaint                 = user32.NewProc("EndPaint")
	loadCursor               = user32.NewProc("LoadCursorW")
	setTimer                 = user32.NewProc("SetTimer")
	killTimer                = user32.NewProc("KillTimer")
	showWindow               = user32.NewProc("ShowWindow")
	updateWindow             = user32.NewProc("UpdateWindow")
	
	textOut                  = gdi32.NewProc("TextOutW")
	setTextColor             = gdi32.NewProc("SetTextColor")
	setBkMode                = gdi32.NewProc("SetBkMode")
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	CW_USEDEFAULT       = 0x80000000
	WM_DESTROY          = 0x0002
	WM_PAINT            = 0x000F
	WM_COMMAND          = 0x0111
	WM_TIMER            = 0x0113
	BS_PUSHBUTTON       = 0x00000000
	WS_CHILD            = 0x40000000
	WS_TABSTOP          = 0x00010000
	SWP_NOZORDER        = 0x0004
	IDC_ARROW           = 32512
	TRANSPARENT         = 1
	CS_HREDRAW          = 0x0002
	CS_VREDRAW          = 0x0001
	COLOR_WINDOW        = 5
	SW_SHOW             = 5
	SW_SHOWNORMAL       = 1
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

type WNDCLASSEX struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type RECT struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

type POINT struct {
	x int32
	y int32
}

type MSG struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      POINT
}

type PAINTSTRUCT struct {
	hdc         uintptr
	fErase      int32
	rcPaint     RECT
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

var (
	hwnd      uintptr
	hwndBtn1  uintptr
	hwndBtn2  uintptr
	hwndBtn3  uintptr
	className = syscall.StringToUTF16Ptr("MemoryCleanerClass")
	autoClean bool
	timerID   uintptr
)

func main() {
	runtime.LockOSThread()
	
	hInstance, _, _ := getModuleHandle.Call(0)
	if hInstance == 0 {
		panic("Failed to get module handle")
	}

	// 加载光标
	hCursor, _, _ := loadCursor.Call(0, uintptr(IDC_ARROW))

	// 注册窗口类
	wc := WNDCLASSEX{
		cbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		style:         CS_HREDRAW | CS_VREDRAW,
		lpfnWndProc:   syscall.NewCallback(wndProc),
		cbClsExtra:    0,
		cbWndExtra:    0,
		hInstance:     hInstance,
		hIcon:         0,
		hCursor:       hCursor,
		hbrBackground: COLOR_WINDOW + 1,
		lpszMenuName:  nil,
		lpszClassName: className,
		hIconSm:       0,
	}
	
	ret, _, _ := registerClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		panic("Failed to register window class")
	}

	// 创建窗口
	windowTitle := syscall.StringToUTF16Ptr("Memory Cleaner - 内存清理工具")
	hwnd, _, _ = createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		WS_OVERLAPPEDWINDOW,
		CW_USEDEFAULT, CW_USEDEFAULT,
		500, 450,
		0, 0, hInstance, 0,
	)
	
	if hwnd == 0 {
		panic("Failed to create window")
	}

	// 显示并更新窗口
	showWindow.Call(hwnd, SW_SHOWNORMAL)
	updateWindow.Call(hwnd)

	// 创建按钮
	btnClass := syscall.StringToUTF16Ptr("BUTTON")
	
	btn1Text := syscall.StringToUTF16Ptr("清理物理内存")
	hwndBtn1, _, _ = createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(btnClass)),
		uintptr(unsafe.Pointer(btn1Text)),
		WS_VISIBLE|WS_CHILD|BS_PUSHBUTTON|WS_TABSTOP,
		50, 220, 150, 40,
		hwnd, 1, hInstance, 0,
	)

	btn2Text := syscall.StringToUTF16Ptr("清理工作集")
	hwndBtn2, _, _ = createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(btnClass)),
		uintptr(unsafe.Pointer(btn2Text)),
		WS_VISIBLE|WS_CHILD|BS_PUSHBUTTON|WS_TABSTOP,
		280, 220, 150, 40,
		hwnd, 2, hInstance, 0,
	)

	btn3Text := syscall.StringToUTF16Ptr("启动自动清理")
	hwndBtn3, _, _ = createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(btnClass)),
		uintptr(unsafe.Pointer(btn3Text)),
		WS_VISIBLE|WS_CHILD|BS_PUSHBUTTON|WS_TABSTOP,
		120, 280, 230, 40,
		hwnd, 3, hInstance, 0,
	)

	// 设置定时器更新显示 (每2秒)
	setTimer.Call(hwnd, 100, 2000, 0)

	// 消息循环
	var msg MSG
	for {
		r, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := beginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

		// 设置背景透明
		setBkMode.Call(hdc, TRANSPARENT)
		setTextColor.Call(hdc, 0x000000) // 黑色文字

		memStatus := getMemoryStatus()
		used := memStatus.ullTotalPhys - memStatus.ullAvailPhys

		lines := []string{
			"=== 内存状态 ===",
			fmt.Sprintf("总物理内存: %s", formatBytes(memStatus.ullTotalPhys)),
			fmt.Sprintf("可用内存: %s", formatBytes(memStatus.ullAvailPhys)),
			fmt.Sprintf("已用内存: %s", formatBytes(used)),
			fmt.Sprintf("使用率: %d%%", memStatus.dwMemoryLoad),
			"",
			"=== 操作说明 ===",
			"点击下方按钮进行内存清理",
		}

		for i, line := range lines {
			text := syscall.StringToUTF16(line)
			textOut.Call(hdc, 20, uintptr(20+i*25), uintptr(unsafe.Pointer(&text[0])), uintptr(len(text)-1))
		}

		endPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0

	case WM_COMMAND:
		switch wParam {
		case 1: // 清理物理内存
			go func() {
				freed := cleanPhysicalMemory()
				title := syscall.StringToUTF16Ptr(fmt.Sprintf("完成 - 释放 %s", formatBytes(freed)))
				setWindowText.Call(hwnd, uintptr(unsafe.Pointer(title)))
				invalidateRect.Call(hwnd, 0, 1)
			}()

		case 2: // 清理工作集
			go func() {
				cleanWorkingSet()
				title := syscall.StringToUTF16Ptr("工作集清理完成")
				setWindowText.Call(hwnd, uintptr(unsafe.Pointer(title)))
				invalidateRect.Call(hwnd, 0, 1)
			}()

		case 3: // 自动清理
			autoClean = !autoClean
			if autoClean {
				timerID, _, _ = setTimer.Call(hwnd, 101, 30000, 0)
				btn3Text := syscall.StringToUTF16Ptr("停止自动清理")
				setWindowText.Call(hwndBtn3, uintptr(unsafe.Pointer(btn3Text)))
				title := syscall.StringToUTF16Ptr("自动清理已启动 (每30秒)")
				setWindowText.Call(hwnd, uintptr(unsafe.Pointer(title)))
			} else {
				killTimer.Call(hwnd, timerID)
				btn3Text := syscall.StringToUTF16Ptr("启动自动清理")
				setWindowText.Call(hwndBtn3, uintptr(unsafe.Pointer(btn3Text)))
				title := syscall.StringToUTF16Ptr("自动清理已停止")
				setWindowText.Call(hwnd, uintptr(unsafe.Pointer(title)))
			}
		}

	case WM_TIMER:
		if wParam == 101 && autoClean {
			go cleanPhysicalMemory()
		}
		invalidateRect.Call(hwnd, 0, 1)

	case WM_DESTROY:
		postQuitMessage.Call(0)
		return 0
	}

	r, _, _ := defWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func cleanPhysicalMemory() uint64 {
	before := getMemoryStatus()

	handle, _, _ := getCurrentProcess.Call()
	setProcessWorkingSetSize.Call(handle, ^uintptr(0), ^uintptr(0))

	emptyWorkingSet := psapi.NewProc("EmptyWorkingSet")
	emptyWorkingSet.Call(handle)

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	after := getMemoryStatus()
	freed := int64(before.ullTotalPhys-before.ullAvailPhys) - int64(after.ullTotalPhys-after.ullAvailPhys)
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
