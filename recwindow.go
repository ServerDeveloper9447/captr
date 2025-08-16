package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"strings"
	"time"

	"github.com/go-toast/toast"
	"golang.design/x/hotkey"
)

func RecordWindow() {
	hwnd := chooseWindow()
	ActivateWindowAndGetBounds(uintptr(hwnd))
	filename := filepath.Join(config.SaveLocation, fmt.Sprintf("Recording_%s.mp4", time.Now().Format("20060102_150405")))
	args := []string{
		"-f", "gdigrab",
		"-framerate", fmt.Sprintf("%d", config.RecordingOpts.FPS),
		"-draw_mouse", ternary(config.RecordingOpts.CaptureMouse, "1", "0"),
		"-show_region", "1",
		"-i", fmt.Sprintf("hwnd=%d", uintptr(hwnd)),
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-profile:v", "main",
		"-level", "4.0",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-y", filename,
	}

	cmd := exec.Command(getFfmpegPath(), args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	stdin, _ := cmd.StdinPipe()
	var start time.Time
	var err error
	if err, start = cmd.Start(), time.Now(); err != nil {
		fmt.Println("Error starting ffmpeg:", err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("\nUnexpected error:", r)
		}
		exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	}()

	var modkeys []hotkey.Modifier
	pressedKeys := map[string]bool{}
	for _, key := range config.HotkeyConfig.Modkeys {
		pressedKeys[key] = true
	}
	for _, mod := range []string{"ctrl", "alt", "shift"} {
		if pressedKeys[mod] {
			switch mod {
			case "ctrl":
				modkeys = append(modkeys, hotkey.ModCtrl)
			case "alt":
				modkeys = append(modkeys, hotkey.ModAlt)
			case "shift":
				modkeys = append(modkeys, hotkey.ModShift)
			}
		}
	}
	fmt.Printf("Recording started. Press %s to stop\n", strings.Join(append(config.HotkeyConfig.Modkeys, config.HotkeyConfig.Finalkey), "+"))
	tickStop := make(chan struct{})
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	i := 0
	last := []string{"🔴", "⚫"}
	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Printf("\r%s Recording time elapsed: %02d:%02d", last[i], int(time.Since(start).Minutes()), int(time.Since(start).Seconds())%60)
				i = (i + 1) % 2
			case <-tickStop:
			}
		}
	}()
	go func() {
		err = cmd.Wait()
		if err != nil {
			fmt.Println("Error waiting for ffmpeg to exit:", err)
		}
	}()
	hk := hotkey.New(modkeys, keys[config.HotkeyConfig.Finalkey])
	err = hk.Register()
	if err != nil {
		fmt.Println("Error registering hotkey:", err)
		return
	}
	defer hk.Unregister()
	keyChan := hk.Keydown()
	for range keyChan {
		tickStop <- struct{}{}
		stdin.Write([]byte("q"))
		stdin.Close()
		fmt.Println("\nStopping recording...")
		notification := toast.Notification{
			AppID:               "Captr",
			Title:               "Recording Stopped",
			Message:             fmt.Sprintf("Recording saved at %s", filename),
			Icon:                filename,
			ActivationArguments: filename,
			Audio:               toast.IM,
			Actions: []toast.Action{
				{Type: "protocol", Label: "Open", Arguments: filename},
			},
		}
		notification.Push()
		break
	}

	fmt.Printf("\nRecording stopped. Recording saved at %s", filename)
}
