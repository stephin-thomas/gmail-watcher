package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"

	"google.golang.org/api/gmail/v1"
)

func get_config_folder() string {
	dirname, _ := os.UserHomeDir()
	var CONFIG_FOLDER string = dirname + "/.config/gmail_watcher/"
	log.Println("Config Folder:-", CONFIG_FOLDER)
	return CONFIG_FOLDER
}
func isAvailable(haystack_array []string, needle string) bool {

	// iterate using the for loop
	for i := 0; i < len(haystack_array); i++ {
		// check
		if haystack_array[i] == needle {
			// return true
			return true
		}
	}
	return false
}
func gen_random_token_name(tokFiles []string, CONFIG_FOLDER *string) []string {
	token_file_name := *CONFIG_FOLDER + "token" + fmt.Sprint(rand.Intn(100)) + ".json"
	for isAvailable(tokFiles, token_file_name) {
		token_file_name = *CONFIG_FOLDER + "token" + fmt.Sprint(rand.Intn(100)) + ".json"
	}
	tokFiles = append(tokFiles, token_file_name)
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
	log.Println("Loading Tokens from ", CONFIG_FOLDER+"Config.json")
	tokFiles, err := load_old_ids(CONFIG_FOLDER + "Config.json")
	if err != nil || append {
		// var sample_tokfile []string
		// tokFiles = sample_tokfile
		fmt.Println("Error loading config.json", err)
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
	save_as_json(tokFiles, CONFIG_FOLDER+"Config.json")
	return tokFiles
}
