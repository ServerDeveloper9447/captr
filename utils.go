package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/JamesHovious/w32"
	"github.com/manifoldco/promptui"
	"golang.design/x/clipboard"
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procEnumWindows = user32.NewProc("EnumWindows")
	procGetWindow   = user32.NewProc("GetWindow")
)

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}
type WindowBounds struct {
	X      int32
	Y      int32
	Width  int32
	Height int32
}

func CopyImgToClipboard(img image.Image) {
	err := clipboard.Init()
	if err != nil {
		fmt.Println("Cannot copy image to clipboard.")
		return
	}
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		fmt.Println("Cannot copy image to clipboard.")
		return
	}
	clipboard.Write(clipboard.FmtImage, buf.Bytes())
	fmt.Println("Image copied to clipboard")
}

func SetProcessDPIAware() error {
	user32 := syscall.NewLazyDLL("user32.dll")
	setProcessDpiAwarenessContext := user32.NewProc("SetProcessDpiAwarenessContext")
	if setProcessDpiAwarenessContext.Find() == nil {
		const DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = ^uintptr(3) + 1 // -4
		ret, _, _ := setProcessDpiAwarenessContext.Call(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)
		if ret != 0 {
			return nil
		}
	}
	shcore := syscall.NewLazyDLL("shcore.dll")
	setProcessDpiAwareness := shcore.NewProc("SetProcessDpiAwareness")
	if setProcessDpiAwareness.Find() == nil {
		const PROCESS_PER_MONITOR_DPI_AWARE = 2
		ret, _, _ := setProcessDpiAwareness.Call(PROCESS_PER_MONITOR_DPI_AWARE)
		if ret == 0 { // S_OK
			return nil
		}
	}
	setProcessDPIAware := user32.NewProc("SetProcessDPIAware")
	ret, _, err := setProcessDPIAware.Call()
	if ret == 0 {
		return fmt.Errorf("SetProcessDPIAware failed: %v", err)
	}

	return nil
}

func GetSystemDPI() (uint32, uint32) {
	user32 := syscall.NewLazyDLL("user32.dll")
	getDC := user32.NewProc("GetDC")
	releaseDC := user32.NewProc("ReleaseDC")

	gdi32 := syscall.NewLazyDLL("gdi32.dll")
	getDeviceCaps := gdi32.NewProc("GetDeviceCaps")

	hdc, _, _ := getDC.Call(0)
	defer releaseDC.Call(0, hdc)

	const LOGPIXELSX = 88
	const LOGPIXELSY = 90

	dpiX, _, _ := getDeviceCaps.Call(hdc, LOGPIXELSX)
	dpiY, _, _ := getDeviceCaps.Call(hdc, LOGPIXELSY)

	return uint32(dpiX), uint32(dpiY)
}

func GetWindowDPI(hwnd uintptr) (uint32, uint32, error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	shcore := syscall.NewLazyDLL("shcore.dll")

	getDpiForWindow := user32.NewProc("GetDpiForWindow")
	if getDpiForWindow.Find() == nil {
		dpi, _, _ := getDpiForWindow.Call(hwnd)
		if dpi != 0 {
			return uint32(dpi), uint32(dpi), nil
		}
	}

	monitorFromWindow := user32.NewProc("MonitorFromWindow")
	getDpiForMonitor := shcore.NewProc("GetDpiForMonitor")

	if monitorFromWindow.Find() == nil && getDpiForMonitor.Find() == nil {
		const MONITOR_DEFAULTTONEAREST = 2
		hMonitor, _, _ := monitorFromWindow.Call(hwnd, MONITOR_DEFAULTTONEAREST)

		if hMonitor != 0 {
			var dpiX, dpiY uint32
			const MDT_EFFECTIVE_DPI = 0

			ret, _, _ := getDpiForMonitor.Call(
				hMonitor,
				MDT_EFFECTIVE_DPI,
				uintptr(unsafe.Pointer(&dpiX)),
				uintptr(unsafe.Pointer(&dpiY)),
			)

			if ret == 0 { // S_OK
				return dpiX, dpiY, nil
			}
		}
	}

	dpiX, dpiY := GetSystemDPI()
	return dpiX, dpiY, nil
}

func GetWindowBounds(hwnd uintptr) (WindowBounds, error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	getWindowRect := user32.NewProc("GetWindowRect")

	var rect RECT
	ret, _, err := getWindowRect.Call(
		hwnd,
		uintptr(unsafe.Pointer(&rect)),
	)

	if ret == 0 {
		return WindowBounds{}, fmt.Errorf("GetWindowRect failed: %v", err)
	}

	bounds := WindowBounds{
		X:      rect.Left,
		Y:      rect.Top,
		Width:  rect.Right - rect.Left,
		Height: rect.Bottom - rect.Top,
	}

	return bounds, nil
}

func GetWindowBoundsWithDPIInfo(hwnd uintptr) (WindowBounds, uint32, uint32, error) {
	bounds, err := GetWindowBounds(hwnd)
	if err != nil {
		return WindowBounds{}, 0, 0, err
	}

	dpiX, dpiY, dpiErr := GetWindowDPI(hwnd)
	if dpiErr != nil {
		return bounds, 96, 96, nil // Default to 96 DPI
	}

	return bounds, dpiX, dpiY, nil
}

func isAltTabWindow(hwnd w32.HWND) bool {
	if !w32.IsWindowVisible(hwnd) {
		return false
	}

	length := len(w32.GetWindowText(hwnd))
	if length == 0 {
		return false
	}

	exStyle := w32.GetWindowLong(hwnd, w32.GWL_EXSTYLE)
	if exStyle&w32.WS_EX_TOOLWINDOW != 0 {
		return false
	}
	ret, _, _ := procGetWindow.Call(uintptr(hwnd), uintptr(4))
	owner := w32.HWND(ret)
	return owner == 0
}

func chooseWindow() w32.HWND {
	windows := make(map[string]w32.HWND)
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		h := w32.HWND(hwnd)
		if isAltTabWindow(h) {
			title := w32.GetWindowText(h)
			windows[fmt.Sprintf("%s - 0x%x", title, h)] = h
		}
		return 1
	})
	_, _, _ = procEnumWindows.Call(cb, 0)

	prompt := promptui.Select{
		Label: "Select Window",
		Items: func() []string {
			values := []string{}
			for k := range windows {
				values = append(values, k)
			}
			return values
		}(),
	}
	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return w32.HWND(0)
	}

	return windows[result]
}

func AllowSetForegroundWindow(processId uint32) error {
	user32 := syscall.NewLazyDLL("user32.dll")
	allowSetForegroundWindow := user32.NewProc("AllowSetForegroundWindow")

	ret, _, err := allowSetForegroundWindow.Call(uintptr(processId))
	if ret == 0 {
		return fmt.Errorf("AllowSetForegroundWindow failed: %v", err)
	}
	return nil
}

// SimulateUserInput simulates minimal user input to allow window activation
func SimulateUserInput() {
	user32 := syscall.NewLazyDLL("user32.dll")
	setForegroundWindow := user32.NewProc("SetForegroundWindow")

	// Get our own window to temporarily focus
	getConsoleWindow := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow")
	consoleHwnd, _, _ := getConsoleWindow.Call()

	if consoleHwnd != 0 {
		setForegroundWindow.Call(consoleHwnd)
	}
}

// BringWindowToTop brings the specified window to the top and activates it
func BringWindowToTop(hwnd uintptr) error {
	user32 := syscall.NewLazyDLL("user32.dll")
	kernel32 := syscall.NewLazyDLL("kernel32.dll")

	isIconic := user32.NewProc("IsIconic")
	showWindow := user32.NewProc("ShowWindow")
	setForegroundWindow := user32.NewProc("SetForegroundWindow")
	getWindowThreadProcessId := user32.NewProc("GetWindowThreadProcessId")
	attachThreadInput := user32.NewProc("AttachThreadInput")
	getCurrentThreadId := kernel32.NewProc("GetCurrentThreadId")
	setWindowPos := user32.NewProc("SetWindowPos")
	bringWindowToTop := user32.NewProc("BringWindowToTop")

	var processId uint32
	targetThreadId, _, _ := getWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processId)))

	AllowSetForegroundWindow(processId)

	ret, _, _ := isIconic.Call(hwnd)
	if ret != 0 {
		const SW_RESTORE = 9
		showWindow.Call(hwnd, SW_RESTORE)
		kernel32.NewProc("Sleep").Call(50)
	}

	currentThreadId, _, _ := getCurrentThreadId.Call()

	var attached bool
	if targetThreadId != currentThreadId && targetThreadId != 0 {
		ret, _, _ := attachThreadInput.Call(currentThreadId, targetThreadId, 1)
		attached = (ret != 0)
	}

	SimulateUserInput()

	// Try to bring window to foreground
	ret, _, _ = setForegroundWindow.Call(hwnd)
	if ret == 0 {
		const HWND_TOPMOST = ^uintptr(0)   // -1
		const HWND_NOTOPMOST = ^uintptr(1) // -2
		const SWP_NOMOVE = 0x0002
		const SWP_NOSIZE = 0x0001
		const SWP_SHOWWINDOW = 0x0040
		setWindowPos.Call(hwnd, HWND_TOPMOST, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW)
		kernel32.NewProc("Sleep").Call(10)

		setWindowPos.Call(hwnd, HWND_NOTOPMOST, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW)
		setForegroundWindow.Call(hwnd)

		bringWindowToTop.Call(hwnd)
	}

	if attached {
		attachThreadInput.Call(currentThreadId, targetThreadId, 0)
	}

	return nil
}

func ForceWindowToTop(hwnd uintptr) error {
	user32 := syscall.NewLazyDLL("user32.dll")
	kernel32 := syscall.NewLazyDLL("kernel32.dll")

	err := BringWindowToTop(hwnd)
	if err == nil {
		return nil
	}

	getWindowThreadProcessId := user32.NewProc("GetWindowThreadProcessId")
	var processId uint32
	getWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processId)))

	keybd_event := user32.NewProc("keybd_event")
	const VK_MENU = 0x12 // Alt key
	const VK_TAB = 0x09  // Tab key
	const KEYEVENTF_KEYUP = 0x0002

	// Simulate Alt+Tab
	keybd_event.Call(VK_MENU, 0, 0, 0)               // Alt down
	keybd_event.Call(VK_TAB, 0, 0, 0)                // Tab down
	keybd_event.Call(VK_TAB, 0, KEYEVENTF_KEYUP, 0)  // Tab up
	keybd_event.Call(VK_MENU, 0, KEYEVENTF_KEYUP, 0) // Alt up

	kernel32.NewProc("Sleep").Call(100)

	// Try to activate the window again
	setForegroundWindow := user32.NewProc("SetForegroundWindow")
	showWindow := user32.NewProc("ShowWindow")

	const SW_SHOW = 5
	const SW_RESTORE = 9

	showWindow.Call(hwnd, SW_RESTORE)
	kernel32.NewProc("Sleep").Call(50)
	showWindow.Call(hwnd, SW_SHOW)
	setForegroundWindow.Call(hwnd)

	return nil
}

func FlashWindowToGetAttention(hwnd uintptr) error {
	user32 := syscall.NewLazyDLL("user32.dll")
	flashWindowEx := user32.NewProc("FlashWindowEx")

	type FLASHWINFO struct {
		cbSize    uint32
		hwnd      uintptr
		dwFlags   uint32
		uCount    uint32
		dwTimeout uint32
	}

	const FLASHW_STOP = 0
	const FLASHW_ALL = 3
	const FLASHW_TIMERNOFG = 12

	flashInfo := FLASHWINFO{
		cbSize:    uint32(unsafe.Sizeof(FLASHWINFO{})),
		hwnd:      hwnd,
		dwFlags:   FLASHW_ALL | FLASHW_TIMERNOFG,
		uCount:    3,
		dwTimeout: 0,
	}

	// Flash the window
	ret, _, err := flashWindowEx.Call(uintptr(unsafe.Pointer(&flashInfo)))
	if ret == 0 {
		return fmt.Errorf("FlashWindowEx failed: %v", err)
	}

	// Stop flashing after bringing to front
	flashInfo.dwFlags = FLASHW_STOP
	flashWindowEx.Call(uintptr(unsafe.Pointer(&flashInfo)))

	// Try to bring to front after flashing
	return BringWindowToTop(hwnd)
}

func ActivateWindowAndGetBounds(hwnd uintptr) (WindowBounds, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")
	sleep := kernel32.NewProc("Sleep")

	getWindowThreadProcessId := user32.NewProc("GetWindowThreadProcessId")
	var processId uint32
	getWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processId)))
	AllowSetForegroundWindow(processId)

	BringWindowToTop(hwnd)
	sleep.Call(200)

	FlashWindowToGetAttention(hwnd)
	sleep.Call(200)

	ForceWindowToTop(hwnd)
	sleep.Call(300)

	bounds, err := GetWindowBounds(hwnd)
	if err != nil {
		return WindowBounds{}, err
	}

	return bounds, nil
}

func getFfmpegPath() string {
	cmd := exec.Command("ffmpeg", "-version")
	if err := cmd.Run(); err == nil {
		return "ffmpeg"
	}

	return filepath.Join(appdataDir, "bin", "ffmpeg.exe")
}
