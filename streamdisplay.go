package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/go-vgo/robotgo"
)

const (
	YOUTUBE_RTMP = "rtmp://a.rtmp.youtube.com/live2"
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
		Options: []string{"Youtube", "Twitch"},
	}, &service, survey.WithValidator(survey.Required))
	if err != nil {
		fmt.Println("Some error occurred")
		return
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
			"-filter_complex", fmt.Sprintf("ddagrab=output_idx=%d:framerate=%d,hwdownload,format=bgra", display, config.RecordingOpts.FPS),
			"-f", "dshow",
			"-i", "audio=\"Stereo Mix (Realtek(R) Audio)\"",
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-b:v", "3000k",
			"-c:a", "aac",
			"-f", "flv",
			fmt.Sprintf("%s/%s", YOUTUBE_RTMP, key),
		}
		cmd := exec.Command(getFfmpegPath(), args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Println("Cannot start ffmpeg")
			os.Exit(0)
		}
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("\nUnexpected error:", r)
			}
			exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
		}()
		err = cmd.Wait()
		if err != nil {
			fmt.Println("Error waiting for ffmpeg to exit:", err)
		}
		fmt.Println("\nStreaming stopped")
	case "Twitch":
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
			"-filter_complex", fmt.Sprintf("ddagrab=output_idx=%d:framerate=%d,hwdownload,format=bgra", display, config.RecordingOpts.FPS),
			"-draw_mouse", ternary(config.RecordingOpts.CaptureMouse, "1", "0"),
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-c:a", "aac",
			"-b:v", "3000k",
			"-f", "flv",
			fmt.Sprintf("%s/%s", TWITCH_RTMP, key),
		}
		cmd := exec.Command(getFfmpegPath(), args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Println("Cannot start ffmpeg")
			os.Exit(0)
		}
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("\nUnexpected error:", r)
			}
			exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
		}()
		err = cmd.Wait()
		if err != nil {
			fmt.Println("Error waiting for ffmpeg to exit:", err)
		}
		fmt.Println("\nStreaming stopped")
	}
}
