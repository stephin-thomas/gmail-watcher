package io_helpers

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
	"github.com/gmail-watcher/exports"
)

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func CreateFolder(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating folder %w", err)
		}
	} else {
		log.Println("Folder exists", path)
	}
	return nil
}
func DeleteFile(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		err := os.Remove(path)
		if err != nil {
			return fmt.Errorf("error deleting file: %v", err)
		}
	} else {
		log.Println("file doesn't exist", path)
	}
	return nil
}

func CopyAssets(sourceFolder string, destinationFolder string) error {
	err1 := filepath.WalkDir(sourceFolder, func(src_path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dest_path := filepath.Join(destinationFolder, filepath.Base(src_path))
		err = CopyFile(src_path, dest_path)
		if err != nil {
			return err
		}
		return nil
	})
	if err1 != nil {
		return fmt.Errorf("error copying assets %w", err1)
	}
	return nil
}

func CopyFile(sourceFile string, destinationFile string) error {
	if !FileExists(destinationFile) {
		file, err := os.Create(destinationFile)
		if err != nil {
			return fmt.Errorf("error creating file %v - %w", destinationFile, err)
		}
		input, err := os.ReadFile(sourceFile)
		if err != nil {
			return fmt.Errorf("error reading file %v - %w", sourceFile, err)
		}
		_, err = file.Write(input)
		if err != nil {
			return fmt.Errorf("error writing to file %v - %w", file.Name(), err)
		}
		return err
	}
	log.Println("files tryping to copy is already found at the dest skipping", sourceFile, destinationFile)
	return nil
}

func Notify(msg string, heading string) error {
	err := beeep.Notify(heading, msg, exports.NOTIFICATION_ICON)
	if err != nil {
		log.Println("error during notification", err)
		return fmt.Errorf("error during notification %w", err)

	}
	err = beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
	if err != nil {
		return fmt.Errorf("error during notification %w", err)
	}
	return nil
}
