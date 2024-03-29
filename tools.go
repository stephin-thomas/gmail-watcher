package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/google/uuid"
	"google.golang.org/api/gmail/v1"
)

func get_config_folder() string {
	var CONFIG_FOLDER string = filepath.Join(xdg.ConfigHome, "gmail_watcher")
	return CONFIG_FOLDER
}

func add_random_token_path(tokFiles *[]string) *string {
	token_file_name := fmt.Sprintf("token_%s.json", uuid.NewString())
	token_file_path := path.Join(CONFIG_FOLDER, token_file_name)
	*tokFiles = append(*tokFiles, token_file_path)
	log.Printf("Token file name generated %v", tokFiles)
	return &token_file_path
}

// func list_difference(new *map[string]struct{}, old *map[string]struct{}) *map[string]struct{} {
// 	diff := map[string]struct{}{}
// 	max_try := 3
// 	try := 1
// 	for new_id := range *new {
// 		if check_if_value_present(new_id, old) {
// 			if try == max_try {
// 				return &diff
// 			}
// 			try += 1
// 		} else {
// 			diff[new_id] = struct{}{}
// 			// diff = append(diff, new_id)
// 		}
// 	}
// 	// defer fmt.Println("Difference is", diff)
// 	return &diff
// }

func create_id_list(records *[]*gmail.Message) *map[string]struct{} {
	id_list := map[string]struct{}{}
	for _, msg := range *records {
		id_list[msg.Id] = struct{}{}
		// id_list = append(id_list, msg.Id) // note the = instead of :=
	}
	return &id_list
}

func add_token(tokFiles *[]string) *string {
	log.Println("Adding new token")
	tok_file_name := add_random_token_path(tokFiles)
	log.Println("Added random token file to:-", tok_file_name)
	serialize_n_save(tokFiles, LOGIN_TOKENS_LIST_FILE)
	return tok_file_name
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
		log.Println("Error:", err)
		// Additional error handling logic can be added here
	}

}
