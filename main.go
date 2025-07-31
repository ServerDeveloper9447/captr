package main

import (
	"fmt"
	"image/png"
	"os"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/vcaesar/screenshot"
)



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
	case 3:
		Screenshot_Display()
	}
}
