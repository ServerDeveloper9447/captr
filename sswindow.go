package main

import (
	"fmt"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-toast/toast"
	"github.com/go-vgo/robotgo"
	"github.com/tailscale/win"
)

func Screenshot_Window() {
	hwnd := chooseWindow()
	var pid uintptr = 0
	user32 := syscall.MustLoadDLL("user32.dll")
	prc := user32.MustFindProc("GetWindowThreadProcessId")
	_, _, _ = prc.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	err := SetProcessDPIAware()
	if err != nil {
		fmt.Printf("Warning: Could not set DPI awareness: %v\n", err)
	}
	robotgo.ActivePid(int(pid))
	robotgo.SetActiveWindow(win.HWND(hwnd))
	ActivateWindowAndGetBounds(uintptr(hwnd))
	bounds, _, _, err := GetWindowBoundsWithDPIInfo(uintptr(hwnd))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	img, err := robotgo.CaptureImg(int(bounds.X), int(bounds.Y), int(bounds.Width), int(bounds.Height))
	if err != nil {
		fmt.Println("Cannot capture screenshot")
		return
	}
	filename := filepath.Join(config.SaveLocation, fmt.Sprintf("Screenshot_%s.png", time.Now().Format("20060102_150405")))
	robotgo.Save(img, filename)
	fmt.Printf("Screenshot saved at %s", filename)
	notification := toast.Notification{
		AppID:               "Captr",
		Title:               "Screenshot Captured",
		Message:             fmt.Sprintf("Screenshot saved at %s", filename),
		Icon:                filename,
		ActivationArguments: filename,
		Audio:               toast.IM,
		Actions: []toast.Action{
			{Type: "protocol", Label: "Open", Arguments: filename},
		},
	}
	notification.Push()
}
