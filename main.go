package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/AlecAivazis/survey/v2"
)

var (
	config         Config
	appdataDir     string
	configFilePath string
	defaultConfig  = Config{
		SaveLocation: filepath.Join(os.Getenv("USERPROFILE"), "Desktop"),
		RecordFunc:   true,
		RecordingOpts: RecordingOptions{
			FPS:          30,
			CaptureMouse: true,
			AudioDevice:  "",
		},
		HotkeyConfig: hotkeyConfig{
			Modkeys:  []string{"ctrl", "shift"},
			Finalkey: "3",
			Note:     "DO NOT CHANGE ANYTHING MANUALLY IN THIS SECTION UNLESS YOU KNOW WHAT YOU'RE DOING. ALWAYS CHANGE THE HOTKEY VIA THE DEFAULT --hotkey FLAG",
		},
	}
)

func extractFFmpegExe(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "bin/ffmpeg.exe") || strings.HasSuffix(f.Name, "bin\\ffmpeg.exe") {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			os.MkdirAll(destDir, 0755)
			outPath := filepath.Join(destDir, "ffmpeg.exe")
			outFile, err := os.Create(outPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, rc)
			return err
		}
	}

	return os.ErrNotExist
}

type hotkeyConfig struct {
	Modkeys  []string `json:"modkeys"`
	Finalkey string   `json:"finalkey"`
	Note     string   `json:"note"`
}

type Config struct {
	SaveLocation  string           `json:"save_location"`
	RecordFunc    bool             `json:"record_func_enabled"`
	RecordingOpts RecordingOptions `json:"recording_options,omitempty"`
	HotkeyConfig  hotkeyConfig     `json:"hotkey_config"`
}

type RecordingOptions struct {
	FPS          int    `json:"fps"`
	CaptureMouse bool   `json:"capture_mouse"`
	AudioDevice  string `json:"audio_device"`
}

func initConfig() {
	var err error
	appdataDir, err = os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	appdataDir = filepath.Join(appdataDir, "captr")
	configFilePath = filepath.Join(appdataDir, ".captr_config.json")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		config = defaultConfig
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			panic(err)
		}
		os.Mkdir(appdataDir, 0755)
		os.WriteFile(configFilePath, data, 0644)
	} else {
		data, err := os.ReadFile(configFilePath)
		if err != nil {
			panic(err)
		}
		var loadedConfig Config
		if err := json.Unmarshal(data, &loadedConfig); err != nil {
			panic(err)
		}

		config = mergeConfig(defaultConfig, loadedConfig)
		mergedData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			panic(err)
		}
		os.WriteFile(configFilePath, mergedData, 0644)
	}
}

func mergeConfig(defaultConfig, loadedConfig Config) Config {
	if loadedConfig.SaveLocation == "" {
		loadedConfig.SaveLocation = defaultConfig.SaveLocation
	}
	if !loadedConfig.RecordFunc {
		loadedConfig.RecordFunc = defaultConfig.RecordFunc
	}
	if loadedConfig.RecordingOpts.FPS == 0 {
		loadedConfig.RecordingOpts.FPS = defaultConfig.RecordingOpts.FPS
	}
	if !loadedConfig.RecordingOpts.CaptureMouse {
		loadedConfig.RecordingOpts.CaptureMouse = defaultConfig.RecordingOpts.CaptureMouse
	}
	if loadedConfig.RecordingOpts.AudioDevice == "" {
		loadedConfig.RecordingOpts.AudioDevice = defaultConfig.RecordingOpts.AudioDevice
	}
	if len(loadedConfig.HotkeyConfig.Modkeys) == 0 {
		loadedConfig.HotkeyConfig.Modkeys = defaultConfig.HotkeyConfig.Modkeys
	}
	if loadedConfig.HotkeyConfig.Finalkey == "" {
		loadedConfig.HotkeyConfig.Finalkey = defaultConfig.HotkeyConfig.Finalkey
	}
	if loadedConfig.HotkeyConfig.Note == "" {
		loadedConfig.HotkeyConfig.Note = defaultConfig.HotkeyConfig.Note
	}

	return loadedConfig
}

func initDownloads() {
	dwnPath := filepath.Join(appdataDir, "bin")
	if _, err := os.Stat(filepath.Join(dwnPath, "ffmpeg.exe")); err == nil {
		return
	}
	if !config.RecordFunc {
		return
	}
	cmd := exec.Command("ffmpeg", "-version")
	if err := cmd.Run(); err == nil {
		return
	}
	fmt.Println("Captr requires ffmpeg to record videos. However, the screenshotting functionality is not affected.")
	var i int
	err := survey.AskOne(&survey.Select{
			Message: "Choose your action",
			Options: []string{
				"Download ffmpeg (Download size: ~148MB, Install size: ~132MB)",
				"Keep only screenshotting functionality",
			},
			Default: "Download ffmpeg (Download size: ~148MB, Install size: ~132MB)",
	}, &i, survey.WithValidator(survey.Required))
	if err != nil {
		fmt.Println("Action Cancelled")
		os.Exit(1)
	}
	if i == 0 {
		resp, err := http.Get("https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n7.1-latest-win64-gpl-7.1.zip")
		if err != nil {
			fmt.Println("Couldn't download ffmpeg")
			os.Exit(1)
		}
		defer resp.Body.Close()
		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			"Downloading ffmpeg",
		)

		out, err := os.Create(filepath.Join(os.TempDir(), "ffmpeg_captr.zip"))
		if err != nil {
			fmt.Println("Couldn't download ffmpeg")
			os.Exit(1)
		}
		defer out.Close()

		_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
		if err != nil {
			fmt.Printf("Couldn't download ffmpeg.")
			os.Exit(1)
		}
		extractFFmpegExe(filepath.Join(os.TempDir(), "ffmpeg_captr.zip"), dwnPath)
		fmt.Printf("FFMPEG has been downloaded to %s", dwnPath)
	} else {
		setConfig("record_func_enabled", false)
	}
}

const HOTKEY_WARNING = "DO NOT CHANGE THESE MANUALLY UNLESS YOU KNOW WHAT YOU'RE DOING"

func init() {
	if !(runtime.GOOS == "windows" && runtime.GOARCH == "amd64") {
		panic("Captr is only supported on Windows x64")
	}
	initConfig()
	configMode, reset, hotkeyConfigMode := flag.Bool("config", false, "Configure Captr"), flag.Bool("reset", false, "Reset Captr and delete appdata"), flag.Bool("hotkey", false, "Register a hotkey for stopping recording")
	flag.Parse()
	if *configMode {
		cmd := exec.Command("notepad.exe", configFilePath)
		if err := cmd.Start(); err != nil {
			fmt.Println("Error starting command:", err)
			return
		}
		os.Exit(0)
	}
	if *reset {
		// Declaring again for safety. Even if anything fails, atleast it won't delete your entire appdata directory
		appdata, _ := os.UserConfigDir()
		var decision bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Are you sure want to reset Captr?",
			Default: false,
		}, &decision, survey.WithValidator(survey.Required))
		if err != nil {
			fmt.Println("Action Aborted")
			os.Exit(0)
		}
		if !decision {
			os.Exit(0)
		}
		err = os.RemoveAll(filepath.Join(appdata, "captr"))
		if err != nil {
			fmt.Println("Couldn't delete appdata directory")
			os.Exit(1)
		}
		fmt.Println("Captr has been reset")
		os.Exit(0)
	}
	if *hotkeyConfigMode {
		mods, finalkey := RegisterHotkey()
		hotkeyConfig := hotkeyConfig{
			Modkeys:  mods,
			Finalkey: finalkey,
			Note:     HOTKEY_WARNING,
		}
		setConfig("hotkey_config", hotkeyConfig)
		fmt.Println("Hotkeys have been registered successfully")
		os.Exit(0)
	}
	initDownloads()
}

func main() {
	fmt.Printf(`
________/\\\\\\\\\__________________________________________________________        
 _____/\\\////////___________________________________________________________       
  ___/\\\/____________________________/\\\\\\\\\______/\\\____________________      
   __/\\\______________/\\\\\\\\\_____/\\\/////\\\__/\\\\\\\\\\\__/\\/\\\\\\\__     
    _\/\\\_____________\////////\\\___\/\\\\\\\\\\__\////\\\////__\/\\\/////\\\_    
     _\//\\\______________/\\\\\\\\\\__\/\\\//////______\/\\\______\/\\\___\///__   
      __\///\\\___________/\\\/////\\\__\/\\\____________\/\\\_/\\__\/\\\_________  
       ____\////\\\\\\\\\_\//\\\\\\\\/\\_\/\\\____________\//\\\\\___\/\\\_________ 
        _______\/////////___\////////\//__\///______________\/////____\///__________

v1.0.1-beta

`)
	fmt.Println("Open config file by passing the --config flag")
	capture_ops := []string{"Record full screen", "Record specific window", "Screenshot specific window", "Screenshot full screen"}
	var i int
	err := survey.AskOne(&survey.Select{
		Message: "Select Action",
		Options: capture_ops,
	}, &i, survey.WithValidator(survey.Required))

	if err != nil {
		fmt.Print("Action Cancelled.")
		return
	}

	switch i {
	case 0:
		RecordDisplay()
	case 2:
		Screenshot_Window()
	case 3:
		Screenshot_Display()
	}
}
