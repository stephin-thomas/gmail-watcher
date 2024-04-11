package helpers

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/gen2brain/beeep"
	"github.com/gmail-watcher/paths"
	"github.com/google/uuid"
	"google.golang.org/api/gmail/v1"
)

func Add_random_token_path(tokFiles *[]string) *string {
	token_file_name := fmt.Sprintf("token_%s.json", uuid.NewString())
	token_file_path := path.Join(paths.CONFIG_FOLDER, token_file_name)
	*tokFiles = append(*tokFiles, token_file_path)
	log.Printf("Token file name generated %v", tokFiles)
	return &token_file_path
}

func Create_id_list(records *[]*gmail.Message) *map[string]struct{} {
	id_list := map[string]struct{}{}
	for _, msg := range *records {
		id_list[msg.Id] = struct{}{}
		// id_list = append(id_list, msg.Id) // note the = instead of :=
	}
	return &id_list
}

func Add_token(tokFiles *[]string) *string {
	log.Println("Adding new token")
	tok_file_name := Add_random_token_path(tokFiles)
	log.Println("Added random token file to:-", tok_file_name)
	Serialize_n_save(tokFiles, paths.LOGIN_TOKENS_LIST_FILE)
	return tok_file_name
}

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
func NotifyEmail(msg *string, user_email *string) {
	err := beeep.Notify(fmt.Sprintf("Gmail Watcher:-%s", *user_email), *msg, paths.NOTIFICATION_ICON)
	if err != nil {
		log.Println("Error during notification", err)
	}
}
