package helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"time"

	"github.com/gmail-watcher/paths"
	"golang.org/x/oauth2"
)

func Change_server_port(creds *oauth2.Config, port int64) {
	server_url := fmt.Sprintf("http://localhost:%d", port)
	if creds.RedirectURL != server_url {
		creds.RedirectURL = server_url
		Serialize_n_save(*creds, paths.CREDENTIALS_FILE)
	}
}

func Load_old_msg_ids(file_name string) (map[string]struct{}, error) {
	var id_list map[string]struct{}
	json_file, err := os.ReadFile(file_name)
	if err != nil {
		return id_list, err
	}
	err2 := json.Unmarshal(json_file, &id_list)
	return id_list, err2

}

func Load_existing_tokens() ([]string, error) {
	var tok_list []string
	json_file, err := os.ReadFile(paths.LOGIN_TOKENS_LIST_FILE)
	if err != nil {
		return nil, err
	} else {
		err2 := json.Unmarshal(json_file, &tok_list)
		return tok_list, err2
	}
}

func Serialize_n_save(json_unser any, file_name string) error {
	json_b, err := json.Marshal(json_unser)
	if err != nil {
		return err
	}
	err = os.WriteFile(file_name, json_b, 0644)
	log.Println("Saved ", file_name)
	return err
}

// Saves a token to a file path.
func SaveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func token_expired(token_expiry *time.Time) bool {
	cur_time := time.Now()
	if cur_time.Sub(*token_expiry).Seconds() >= 0 {
		return true
	} else {
		return false
	}
}

// Retrieves a token from a local file.
func TokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}
