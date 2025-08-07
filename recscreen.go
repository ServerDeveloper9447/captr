package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"

	"strings"
	"time"

	"github.com/go-toast/toast"
	"github.com/go-vgo/robotgo"
	"github.com/manifoldco/promptui"
	"golang.design/x/hotkey"
)

var (
	keys = map[string]hotkey.Key{
		"a": hotkey.KeyA,
		"b": hotkey.KeyB,
		"c": hotkey.KeyC,
		"d": hotkey.KeyD,
		"e": hotkey.KeyE,
		"f": hotkey.KeyF,
		"g": hotkey.KeyG,
		"h": hotkey.KeyH,
		"i": hotkey.KeyI,
		"j": hotkey.KeyJ,
		"k": hotkey.KeyK,
		"l": hotkey.KeyL,
		"m": hotkey.KeyM,
		"n": hotkey.KeyN,
		"o": hotkey.KeyO,
		"p": hotkey.KeyP,
		"q": hotkey.KeyQ,
		"r": hotkey.KeyR,
		"s": hotkey.KeyS,
		"t": hotkey.KeyT,
		"u": hotkey.KeyU,
		"v": hotkey.KeyV,
		"w": hotkey.KeyW,
		"x": hotkey.KeyX,
		"y": hotkey.KeyY,
		"z": hotkey.KeyZ,
		"0": hotkey.Key0,
		"1": hotkey.Key1,
		"2": hotkey.Key2,
		"3": hotkey.Key3,
		"4": hotkey.Key4,
		"5": hotkey.Key5,
		"6": hotkey.Key6,
		"7": hotkey.Key7,
		"8": hotkey.Key8,
		"9": hotkey.Key9,
	}
)

func RecordDisplay() {
	active_displays := robotgo.DisplaysNum()
	displays := []string{"Display 1 (Primary)"}
	for i := 2; i < active_displays; i++ {
		displays = append(displays, fmt.Sprintf("Display %d", i))
	}
	prompt := promptui.Select{
		Label: "Select Display",
		Items: displays,
	}

	display, _, err := prompt.Run()
	if err != nil {
		fmt.Print("Some error occurred")
		return
	}

	x, y, w, h := robotgo.GetDisplayBounds(display)
	filename := filepath.Join(config.SaveLocation, fmt.Sprintf("Recording_Disp%d_%dx%d_%s.mp4", display+1, w, h, time.Now().Format("20060102_150405")))
	args := []string{
		"-f", "gdigrab",
		"-framerate", fmt.Sprintf("%d", config.RecordingOpts.FPS),
		"-offset_x", strconv.Itoa(x),
		"-offset_y", strconv.Itoa(y),
		"-video_size", fmt.Sprintf("%dx%d", w, h),
		"-draw_mouse", ternary(config.RecordingOpts.CaptureMouse, "1", "0"),
		"-i", "desktop",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-profile:v", "main",
		"-level", "4.0",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-movflags", "+faststart",
		"-y", filename,
	}

	cmd := exec.Command(getFfmpegPath(), args...)
	stdin, _ := cmd.StdinPipe()
	var start time.Time
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

	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for ffmpeg to exit:", err)
	}
	fmt.Printf("\nRecording stopped. Recording saved at %s", filename)
}
