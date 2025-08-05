package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/manifoldco/promptui"
	"golang.design/x/hotkey"
)

type RecordingOptions struct {
	FPS          int    `json:"fps"`
	CaptureMouse bool   `json:"capture_mouse"`
	AudioDevice  string `json:"audio_device"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

var defaultOpts = map[string]any{
	"fps":           30,
	"capture_mouse": true,
	"audio_device":  "",
}

func mergeRecordingDefaults() {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		initConfig()
		return
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Println("Error reading config:", err)
		return
	}

	recRaw, ok := config["recording_options"]
	if !ok {
		config["recording_options"] = defaultOpts
	} else {
		recMap, ok := recRaw.(map[string]any)
		if !ok {
			fmt.Println("Error reading recording options")
			return
		}

		changed := false
		for k, v := range defaultOpts {
			if _, exists := recMap[k]; !exists {
				recMap[k] = v
				changed = true
			}
		}

		if changed {
			config["recording_options"] = recMap
		} else {
			return
		}
	}

	out, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configFilePath, out, 0644)
}

func ternary(cond bool, a, b any) any {
	if cond {
		return a
	}
	return b
}

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
		"-draw_mouse", ternary(config.RecordingOpts.CaptureMouse, "1", "0").(string),
		"-i", "desktop",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-y", filename,
	}

	cmd := exec.Command(getFfmpegPath(), args...)
	stdin, _ := cmd.StdinPipe()
	var start time.Time
	if err, start = cmd.Start(), time.Now(); err != nil {
		fmt.Println("Error starting ffmpeg:", err)
		return
	}

	fmt.Println("Recording started. Press ctrl+shift+3 to stop")
	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			select {
			case <-ticker.C:
				fmt.Printf("\rRecording time elapsed: %02d:%02d", int(time.Since(start).Minutes()), int(time.Since(start).Seconds())%60)
			default:
			}
		}
		fmt.Println()
	}()
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.Key3)
	err = hk.Register()
	if err != nil {
		fmt.Println("Error registering hotkey:", err)
		return
	}
	defer hk.Unregister()
	keyChan := hk.Keydown()
	for range keyChan {
		fmt.Println("\nStopping recording...")
		stdin.Write([]byte("q"))
		stdin.Close()
		ticker.Stop()
		break
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for ffmpeg to exit:", err)
	}
	fmt.Printf("\nRecording stopped. Recording saved at %s", filename)
}
