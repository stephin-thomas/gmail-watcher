package io_helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2"
)

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
func Load_json_list(path string) ([]string, error) {
	var tok_list []string
	json_file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	} else {
		err2 := json.Unmarshal(json_file, &tok_list)
		return tok_list, err2
	}
}
