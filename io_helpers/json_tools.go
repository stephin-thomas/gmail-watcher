package io_helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2"
)

func SerializeNsave(json_obj any, file_name string) error {
	json_b, err := json.Marshal(json_obj)
	if err != nil {
		log.Printf("Marshaling json error %s\n", file_name)
		return err
	}
	err = os.WriteFile(file_name, json_b, 0644)
	if err != nil {
		log.Printf("Unable to save json :- %s\n", file_name)
	}
	log.Println("saved serialised data to ", file_name)
	return err
}

// Saves a token to a file path.
func SaveToken(file_path string, token *oauth2.Token) error {
	log.Printf("Saving credential file to: %s\n", file_path)
	f, err := os.OpenFile(file_path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		return fmt.Errorf("json encoding error: %w", err)
	}
	fmt.Printf("Login added")
	return nil
}

// Retrieves a token from a local file.
func DeserializeFromFile(file_path string, output any) error {
	f, err := os.Open(file_path)
	if err != nil {
		return err
	}
	defer f.Close()
	// tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(output)
	if err != nil {
		return err
	}
	return nil
}
