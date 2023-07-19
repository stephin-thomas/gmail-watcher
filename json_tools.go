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

func change_server_port(CONFIG_FOLDER *string, port int64) {
	credentials_file := *CONFIG_FOLDER + "credentials.json"
	creds, err := load_creds(credentials_file)
	if err != nil {
		log.Printf("Error loading credentials.json %v", err)
	}
	server_url := fmt.Sprintf("http://localhost:%d", port)
	if creds.Installed.RedirectURIs[0] != server_url {
		creds.Installed.RedirectURIs[0] = server_url
	}
	save_as_json(creds, credentials_file)
}

type Credentials struct {
	Installed Fields `json:"installed"`
}
type Fields struct {
	AuthProvider string   `json:"auth_provider_x509_cert_url"`
	AuthUri      string   `json:"auth_uri"`
	TokenUri     string   `json:"token_uri"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	ProjectID    string   `json:"project_id"`
	RedirectURIs []string `json:"redirect_uris"`
}

func load_creds(file_name string) (Credentials, error) {
	var cred_json Credentials
	json_file, err := os.ReadFile(file_name)
	if err != nil {
		log.Fatalln("Unable to load credentials.json", err)
	}
	err2 := json.Unmarshal(json_file, &cred_json)
	return cred_json, err2

}
func load_old_ids(file_name string) ([]string, error) {
	var id_list []string
	json_file, err := os.ReadFile(file_name)
	if err != nil {
		return id_list, err
	}
	err2 := json.Unmarshal(json_file, &id_list)
	return id_list, err2

}

func save_as_json(id_list any, file_name string) error {
	json_b, err := json.Marshal(id_list)
	if err != nil {
		return err
	}
	err = os.WriteFile(file_name, json_b, 0644)
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
