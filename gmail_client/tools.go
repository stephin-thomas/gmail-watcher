package gmail_client

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/gmail-watcher/io_helpers"
	"github.com/gmail-watcher/paths"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
)

func Add_random_token_path(tokFiles *[]string) *string {
	token_file_name := fmt.Sprintf("token_%s.json", uuid.NewString())
	token_file_path := path.Join(paths.CONFIG_FOLDER, token_file_name)
	*tokFiles = append(*tokFiles, token_file_path)
	log.Printf("Token file name generated %v", tokFiles)
	return &token_file_path
}

func CreateIDList(records *[]*gmail.Message) *map[string]struct{} {
	id_list := map[string]struct{}{}
	for _, msg := range *records {
		id_list[msg.Id] = struct{}{}
		// id_list = append(id_list, msg.Id) // note the = instead of :=
	}
	return &id_list
}

func Add_token(tokFiles *[]string) (*string, error) {
	log.Println("Adding new token")
	tok_file_name := Add_random_token_path(tokFiles)
	log.Println("Added random token file to:-", tok_file_name)
	err := io_helpers.Serialize_n_save(tokFiles, paths.LOGIN_TOKENS_LIST_FILE)
	if err != nil {
		return nil, fmt.Errorf("Error adding tokens:- %w", err)
	}
	return tok_file_name, nil
}
func Change_server_port(creds *oauth2.Config, port int64) error {
	server_url := fmt.Sprintf("http://localhost:%d", port)
	if creds.RedirectURL != server_url {
		creds.RedirectURL = server_url
		err := io_helpers.Serialize_n_save(*creds, paths.CREDENTIALS_FILE)
		return fmt.Errorf("Error changing server port:- %w", err)
	}
	return nil
}

func LoadIDList(file_name string) (map[string]struct{}, error) {
	var id_list map[string]struct{}
	json_file, err := os.ReadFile(file_name)
	if err != nil {
		return id_list, err
	}
	err2 := json.Unmarshal(json_file, &id_list)
	return id_list, err2

}

func token_expired(token_expiry *time.Time) bool {
	cur_time := time.Now()
	if cur_time.Sub(*token_expiry).Seconds() >= 0 {
		return true
	} else {
		return false
	}
}
