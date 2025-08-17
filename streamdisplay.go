package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/go-toast/toast"
	"github.com/go-vgo/robotgo"
	"golang.design/x/hotkey"
)

const (
	YOUTUBE_RTMP = "rtmp://x.rtmp.youtube.com/live2"
	TWITCH_RTMP  = "rtmp://ingest.global-contribute.live-video.net/app"
)

func StreamDisp() {
	active_displays := robotgo.DisplaysNum()
	displays := []string{"Display 1 (Primary)"}
	for i := 2; i < active_displays; i++ {
		displays = append(displays, fmt.Sprintf("Display %d", i))
	}
	var display int
	err := survey.AskOne(&survey.Select{
		Message: "Select Display",
		Options: displays,
	}, &display, survey.WithValidator(survey.Required))
	if err != nil {
		fmt.Print("Some error occurred")
		return
	}

	// _, _, w, h := robotgo.GetDisplayBounds(display)
	var service string
	err = survey.AskOne(&survey.Select{
		Message: "Select Service",
		Options: []string{"Youtube", "Twitch", "Twitch (Test Stream)"},
	}, &service, survey.WithValidator(survey.Required))
	if err != nil {
		fmt.Println("Some error occurred")
		return
	}

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

	switch service {
	case "Youtube":
		key := config.StreamConfig.YoutubeStreamKey
		if key == "" {
			fmt.Println("Stream key not set.")
			err = survey.AskOne(&survey.Password{
				Message: "Enter your stream key for YouTube:",
			}, &key, survey.WithValidator(survey.Required))
			if err != nil {
				fmt.Println("Error asking for prompt")
				return
			}
			stream_config := config.StreamConfig
			stream_config.YoutubeStreamKey = key
			setConfig("stream_config", stream_config)
		}
		args := []string{
			"-filter_complex", fmt.Sprintf("ddagrab=output_idx=%d:framerate=%d:draw_mouse=%d,hwdownload,format=bgra", display, config.RecordingOpts.FPS, ternary(config.RecordingOpts.CaptureMouse, 1, 0)),
			"-f", "dshow",
			"-i", fmt.Sprintf("audio=%s", config.RecordingOpts.AudioDevice),
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-b:v", "3000k",
			"-c:a", "aac",
			"-f", "flv",
			fmt.Sprintf("%s/%s", YOUTUBE_RTMP, key),
		}
		fmt.Println("./ffmpeg", strings.Join(args, " "))
		cmd := exec.Command(getFfmpegPath(), args...)
		stdin, _ := cmd.StdinPipe()
		var start time.Time
		if err, start = cmd.Start(), time.Now(); err != nil {
			fmt.Println("Cannot start ffmpeg")
			os.Exit(0)
		}
		fmt.Printf("Streaming started on youtube. Press %s to stop\n", strings.Join(append(config.HotkeyConfig.Modkeys, config.HotkeyConfig.Finalkey), "+"))
		tickStop := make(chan struct{})
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		i := 0
		last := []string{"🔴", "⚫"}
		go func() {
			for {
				select {
				case <-ticker.C:
					fmt.Printf("\r%s Streaming time elapsed: %02d:%02d", last[i], int(time.Since(start).Minutes()), int(time.Since(start).Seconds())%60)
					i = (i + 1) % 2
				case <-tickStop:
				}
			}
		}()
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("\nUnexpected error:", r)
			}
			exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
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
			fmt.Println("\nStopping streaming...")
			notif := toast.Notification{
				AppID: "Captr",
				Title: "Streaming stopped",
			}
			notif.Push()
			break
		}
		fmt.Println("\nStreaming stopped")
	case "Twitch", "Twitch (Test Stream)":
		key := config.StreamConfig.TwitchStreamKey
		if key == "" {
			fmt.Println("Stream key not set")
			err := survey.AskOne(&survey.Password{
				Message: "Enter your stream key for Twitch:",
			}, &key, survey.WithValidator(survey.Required))
			if err != nil {
				fmt.Println("Error asking for prompt")
				return
			}
			stream_config := config.StreamConfig
			stream_config.TwitchStreamKey = key
			setConfig("stream_config", stream_config)
		}
		args := []string{
			"-filter_complex", fmt.Sprintf("ddagrab=output_idx=%d:framerate=%d:draw_mouse=%d,hwdownload,format=bgra,format=yuv420p", display, config.RecordingOpts.FPS, ternary(config.RecordingOpts.CaptureMouse, 1, 0)),
			"-f", "dshow",
			"-i", fmt.Sprintf("audio=%s", config.RecordingOpts.AudioDevice),
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-b:v", "3000k",
			"-c:a", "aac",
			"-pix_fmt", "yuv420p",
			"-f", "flv",
			fmt.Sprintf("%s/%s", TWITCH_RTMP, ternary(service == "Twitch (Test Stream)", fmt.Sprintf("%s?bandwidthtest=true", key), key)),
		}
		cmd := exec.Command(getFfmpegPath(), args...)
		stdin, _ :=  cmd.StdinPipe()
		var start time.Time
		if err, start = cmd.Start(), time.Now(); err != nil {
			fmt.Println("Cannot start ffmpeg")
			os.Exit(0)
		}
		fmt.Printf("Streaming started on twitch. Press %s to stop\n", strings.Join(append(config.HotkeyConfig.Modkeys, config.HotkeyConfig.Finalkey), "+"))
		tickStop := make(chan struct{})
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		i := 0
		last := []string{"🔴", "⚫"}
		go func() {
			for {
				select {
				case <-ticker.C:
					fmt.Printf("\r%s Streaming time elapsed: %02d:%02d", last[i], int(time.Since(start).Minutes()), int(time.Since(start).Seconds())%60)
					i = (i + 1) % 2
				case <-tickStop:
				}
			}
		}()
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("\nUnexpected error:", r)
			}
			exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
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
			fmt.Println("\nStopping streaming...")
			notif := toast.Notification{
				AppID: "Captr",
				Title: "Streaming stopped",
			}
			notif.Push()
			break
		}
		fmt.Println("\nStreaming stopped")
	}
}
