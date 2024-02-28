package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"

	"time"

	"golang.org/x/oauth2"
)

func change_server_port(creds *oauth2.Config, port int64) {
	server_url := fmt.Sprintf("http://localhost:%d", port)
	if creds.RedirectURL != server_url {
		creds.RedirectURL = server_url
		serialize_n_save(*creds, CREDENTIALS_FILE)
	}
}

func load_old_msg_ids(file_name string) (map[string]struct{}, error) {
	var id_list map[string]struct{}
	json_file, err := os.ReadFile(file_name)
	if err != nil {
		return id_list, err
	}
	err2 := json.Unmarshal(json_file, &id_list)
	return id_list, err2

}

func load_existing_tokens() ([]string, error) {
	var tok_list []string
	json_file, err := os.ReadFile(LOGIN_TOKENS_LIST_FILE)
	if err != nil {
		return nil, err
	} else {
		err2 := json.Unmarshal(json_file, &tok_list)
		return tok_list, err2
	}
}

func (c clientService) save() error {
	err := serialize_n_save(c.id_db, c.db)
	return err
}

func serialize_n_save(json_unser any, file_name string) error {
	json_b, err := json.Marshal(json_unser)
	if err != nil {
		return err
	}
	err = os.WriteFile(file_name, json_b, 0644)
	log.Println("Saved ", file_name)
	return err
}

func create_folder(path string) {
	_, err := os.Stat(path)
	if err != nil {
		err := os.Mkdir(path, fs.ModePerm)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
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
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}
