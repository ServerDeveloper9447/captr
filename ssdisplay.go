package main

import (
	"fmt"
	"image"
	"path/filepath"
	"time"

	"github.com/go-toast/toast"
	"github.com/go-vgo/robotgo"
	"github.com/manifoldco/promptui"
)

func takeScreenshot(fileName string, displayNum int) image.Image {
	robotgo.DisplayID = displayNum
	img, err := robotgo.CaptureImg()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	robotgo.Save(img, fileName)
	return img
}

func Screenshot_Display() {
	active_displays := robotgo.DisplaysNum()
	if active_displays == 1 {
		_, _, w, h := robotgo.GetDisplayBounds(0)
		fileName := filepath.Join(config.SaveLocation, fmt.Sprintf("Screenshot_%s_%dx%d.png", time.Now().Format("20060102_150405"), w, h))
		takeScreenshot(fileName, 0)
		fmt.Printf("Screenshot saved at %s", fileName)
		notification := toast.Notification{
			AppID:               "Captr",
			Title:               "Screenshot Captured",
			Message:             fmt.Sprintf("Screenshot saved at %s", fileName),
			Icon:                fileName,
			ActivationArguments: fileName,
			Audio:               toast.IM,
			Actions: []toast.Action{
				{Type: "protocol", Label: "Open", Arguments: fileName},
			},
		}
		notification.Push()
		return
	}

	displays := []string{"Screenshot all displays", "Display 1 (Primary)"}
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

	switch display {
	case 0:
		for i := range active_displays {
			_, _, w, h := robotgo.GetDisplayBounds(i)
			fileName := filepath.Join(config.SaveLocation, fmt.Sprintf("Screenshot_Disp%d_%s_%dx%d.png", i+1, time.Now().Format("20060102_150405"), w, h))
			takeScreenshot(fileName, i)
			fmt.Printf("Screenshot saved at %s", fileName)
		}
		notification := toast.Notification{
			AppID:               "Captr",
			Title:               fmt.Sprintf("%d Screenshot(s) Captured", active_displays),
			Message:             fmt.Sprintf("Screenshot saved at %s", config.SaveLocation),
			ActivationArguments: config.SaveLocation,
			Audio:               toast.IM,
			Actions: []toast.Action{
				{Type: "protocol", Label: "Open Folder", Arguments: config.SaveLocation},
			},
		}
		notification.Push()
	case 1:
		_, _, w, h := robotgo.GetDisplayBounds(0)
		fileName := filepath.Join(config.SaveLocation, fmt.Sprintf("Screenshot_Disp%d_%s_%dx%d.png", 1, time.Now().Format("20060102_150405"), w, h))
		takeScreenshot(fileName, 0)
		fmt.Printf("Screenshot saved at %s", fileName)
		notification := toast.Notification{
			AppID:               "Captr",
			Title:               "Screenshot Captured",
			Message:             fmt.Sprintf("Screenshot saved at %s", fileName),
			Icon:                fileName,
			ActivationArguments: fileName,
			Audio:               toast.IM,
			Actions: []toast.Action{
				{Type: "protocol", Label: "Open", Arguments: fileName},
			},
		}
		notification.Push()
	default:
		_, _, w, h := robotgo.GetDisplayBounds(display - 1)
		fileName := filepath.Join(config.SaveLocation, fmt.Sprintf("Screenshot_Disp%d_%s_%dx%d.png", display, time.Now().Format("20060102_150405"), w, h))
		takeScreenshot(fileName, display-1)
		fmt.Printf("Screenshot saved at %s", fileName)
		notification := toast.Notification{
			AppID:               "Captr",
			Title:               fmt.Sprintf("Screenshot Captured of Display %d", display),
			Message:             fmt.Sprintf("Screenshot saved at %s", fileName),
			Icon:                fileName,
			ActivationArguments: fileName,
			Audio:               toast.IM,
			Actions: []toast.Action{
				{Type: "protocol", Label: "Open", Arguments: fileName},
			},
		}
		notification.Push()
	}
}
