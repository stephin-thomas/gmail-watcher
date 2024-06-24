package io_helpers

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
	"github.com/gmail-watcher/paths"
)

func Create_folder(path string) {
	_, err := os.Stat(path)
	if err != nil {
		err := os.Mkdir(path, fs.ModePerm)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}
}
func Copy_asset(sourceFile string, destinationFile string) {
	input, err1 := os.ReadFile(sourceFile)
	err2 := os.MkdirAll(filepath.Dir(destinationFile), os.ModePerm)
	if _, err := os.Stat(destinationFile); os.IsNotExist(err) {
		file, err1 := os.Create(destinationFile)
		_, err2 := file.Write(input)
		//err = os.WriteFile(file, input, 0644)
		if err1 != nil || err2 != nil {
			log.Println(err)
			log.Println("Error creating", destinationFile)
			return
		}

	}
	err := errors.Join(err1, err2)

	// Handle the combined error
	if err != nil {
		log.Println("Error:", err)
		// Additional error handling logic can be added here
	}

}
func Notify(msg *string, user_email *string) {
	err := beeep.Notify(fmt.Sprintf("Gmail Watcher:-%s", *user_email), *msg, paths.NOTIFICATION_ICON)
	if err != nil {
		log.Println("Error during notification", err)
	}
	err = beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
	if err != nil {
		panic(err)
	}
}
