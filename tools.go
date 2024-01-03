package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"

	"github.com/adrg/xdg"
	"google.golang.org/api/gmail/v1"
)

var CONFIG_ID_FILE string = path.Join(CONFIG_FOLDER, "Config.json")

func get_config_folder() string {
	var CONFIG_FOLDER string = filepath.Join(xdg.ConfigHome, "gmail_watcher")
	log.Println("Config Folder:-", CONFIG_FOLDER)
	return CONFIG_FOLDER
}
func isAvailable(haystack_array []string, needle string) bool {
	for i := 0; i < len(haystack_array); i++ {
		if haystack_array[i] == needle {
			return true
		}
	}
	return false
}
func gen_random_token_name(tokFiles []string, CONFIG_FOLDER *string) []string {
	token_file_name := "token" + fmt.Sprint(rand.Intn(100)) + ".json"
	token_file_path := path.Join(*CONFIG_FOLDER, token_file_name)
	for isAvailable(tokFiles, token_file_path) {
		return gen_random_token_name(tokFiles, CONFIG_FOLDER)
	}
	tokFiles = append(tokFiles, token_file_path)
	log.Printf("Token file name generated %v", tokFiles)
	return tokFiles
}
func check_if_value_present(needle string, haystack *[]string) bool {
	for _, entry := range *haystack {
		if needle == entry {
			return true
		}
	}
	return false

}
func list_difference(new []string, old []string) []string {
	var diff []string

	max_try := 3
	try := 1
	for _, new_id := range new {
		if check_if_value_present(new_id, &old) {
			if try == max_try {
				return diff
			}
			try += 1
		} else {
			diff = append(diff, new_id)
		}
	}
	return diff
}

func create_id_list(records []*gmail.Message) []string {
	var id_list []string
	for _, msg := range records {
		id_list = append(id_list, msg.Id) // note the = instead of :=
	}
	return id_list
}

func load_token_files(CONFIG_FOLDER string, append bool) []string {
	log.Println("Loading Tokens from ", CONFIG_ID_FILE)
	tokFiles, err := load_old_ids(CONFIG_ID_FILE)
	if err != nil || append {
		// var sample_tokfile []string
		// tokFiles = sample_tokfile
		fmt.Println("Error loading config.json\n pass --login to login", err)
		tokFiles = add_token(tokFiles, CONFIG_FOLDER)
	} else {
		log.Println("Previously existing tokens", tokFiles)
	}
	return tokFiles

}

func add_token(tokFiles []string, CONFIG_FOLDER string) []string {
	log.Println("Adding new token")
	tokFiles = gen_random_token_name(tokFiles, &CONFIG_FOLDER)
	log.Println("Added random token file as:-", tokFiles)
	save_as_json(tokFiles, CONFIG_ID_FILE)
	return tokFiles
}

func copy_asset(sourceFile string, destinationFile string) {
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
		fmt.Println("Error:", err)
		// Additional error handling logic can be added here
	}

}
