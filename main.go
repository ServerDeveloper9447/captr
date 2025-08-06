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

	"github.com/manifoldco/promptui"
	"github.com/schollz/progressbar/v3"
)

var (
	config         Config
	appdataDir     string
	configFilePath string
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
	SaveLocation  string            `json:"save_location"`
	RecordFunc    bool              `json:"record_func_enabled"`
	RecordingOpts *RecordingOptions `json:"recording_options,omitempty"`
	HotkeyConfig  *hotkeyConfig     `json:"hotkey_config"`
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
		home, _ := os.UserHomeDir()
		config = Config{
			SaveLocation:  filepath.Join(home, "Desktop"),
			RecordFunc:    true,
			RecordingOpts: nil,
			HotkeyConfig:  nil,
		}
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
		if err := json.Unmarshal(data, &config); err != nil {
			panic(err)
		}
	}
}

func initDownloads() {
	dwnPath := filepath.Join(appdataDir, "bin")
	if _, err := os.Stat(filepath.Join(dwnPath, "ffmpeg.exe")); err == nil {
		mergeRecordingDefaults()
		return
	}
	if !config.RecordFunc {
		return
	}
	cmd := exec.Command("ffmpeg", "-version")
	if err := cmd.Run(); err == nil {
		mergeRecordingDefaults()
		return
	}
	fmt.Println("Captr requires ffmpeg to record videos. However, the screenshotting functionality is not affected.")
	prompt := promptui.Select{
		Label: "Choose your action",
		Items: []string{
			"Download ffmpeg (Download size: ~148MB, Install size: ~132MB)",
			"Keep only screenshotting functionality",
		},
	}
	i, _, err := prompt.Run()
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
		mergeRecordingDefaults()
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
		prompt := promptui.Prompt{
			Label:     "Are you sure you want to reset Captr",
			IsConfirm: true,
		}
		_, err := prompt.Run()
		if err != nil {
			fmt.Println("Action Aborted")
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
			Modkeys: mods,
			Finalkey: finalkey,
			Note:  HOTKEY_WARNING,
		}
		setConfig("hotkey_config", hotkeyConfig)
		fmt.Println("Hotkeys have been registered successfully")
		os.Exit(0)
	}
	initDownloads()
}

func main() {
	fmt.Println(`
________/\\\\\\\\\__________________________________________________________        
 _____/\\\////////___________________________________________________________       
  ___/\\\/____________________________/\\\\\\\\\______/\\\____________________      
   __/\\\______________/\\\\\\\\\_____/\\\/////\\\__/\\\\\\\\\\\__/\\/\\\\\\\__     
    _\/\\\_____________\////////\\\___\/\\\\\\\\\\__\////\\\////__\/\\\/////\\\_    
     _\//\\\______________/\\\\\\\\\\__\/\\\//////______\/\\\______\/\\\___\///__   
      __\///\\\___________/\\\/////\\\__\/\\\____________\/\\\_/\\__\/\\\_________  
       ____\////\\\\\\\\\_\//\\\\\\\\/\\_\/\\\____________\//\\\\\___\/\\\_________ 
        _______\/////////___\////////\//__\///______________\/////____\///__________

	`)
	fmt.Println("Open config file by passing the --config flag")
	capture_ops := []string{"Record full screen", "Record specific window", "Screenshot specific window", "Screenshot full screen"}
	prompt := promptui.Select{
		Label:        "Select Action",
		Items:        capture_ops,
		HideSelected: true,
	}

	i, _, err := prompt.Run()
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
