package main

import (
	"fmt"
	"image/png"
	"os"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/vcaesar/screenshot"
)

func TakeScreenshot(fileName string, displayNum int) {
	img, err := screenshot.CaptureDisplay(displayNum)
	if err != nil {
		fmt.Print("Error: ", err)
		return
	}
	file, _ := os.Create(fileName)
	defer file.Close()
	png.Encode(file, img)
}

func Screenshot_Display() {
	userHomeDir, _ := os.UserHomeDir()
	active_displays := screenshot.NumActiveDisplays()
	if active_displays == 1 {
		bounds := screenshot.GetDisplayBounds(0)
		fileName := fmt.Sprintf("%s\\Desktop\\Screenshot_%s_%dx%d.png", userHomeDir, time.Now().Format("20060102_150405"), bounds.Dx(), bounds.Dy())
		TakeScreenshot(fileName, 0)
		fmt.Printf("Screenshot saved at %s", fileName)
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
	}
	switch display {
	case 0:
		for i := range active_displays {
			bounds := screenshot.GetDisplayBounds(0)
			fileName := fmt.Sprintf("%s\\Desktop\\Screenshot_Disp%d_%s_%dx%d.png", userHomeDir, i, time.Now().Format("20060102_150405"), bounds.Dx(), bounds.Dy())
			TakeScreenshot(fileName, i)
			fmt.Printf("Screenshot saved at %s", fileName)
		}
	case 1:
		bounds := screenshot.GetDisplayBounds(0)
		fileName := fmt.Sprintf("%s\\Desktop\\Screenshot_%s_%dx%d.png", userHomeDir, time.Now().Format("20060102_150405"), bounds.Dx(), bounds.Dy())
		TakeScreenshot(fileName, 0)
		fmt.Printf("Screenshot saved at %s", fileName)
	default:
		bounds := screenshot.GetDisplayBounds(display - 1)
		fileName := fmt.Sprintf("%s\\Desktop\\Screenshot_Disp%d_%s_%dx%d.png", userHomeDir, display, time.Now().Format("20060102_150405"), bounds.Dx(), bounds.Dy())
		TakeScreenshot(fileName, display-1)
		fmt.Printf("Screenshot saved at %s", fileName)
	}
}
