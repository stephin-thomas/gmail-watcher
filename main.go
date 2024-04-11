package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gmail-watcher/daemon"
	"github.com/gmail-watcher/gmail_client"
	"github.com/gmail-watcher/helpers"
	"github.com/gmail-watcher/paths"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

func main() {
	logFile, err := os.OpenFile(paths.LOG_FILE_PATH, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Redirect log output to the file
	log.SetOutput(logFile)
	log.Println("Config Folder:-", paths.CONFIG_FOLDER)
	args := os.Args
	if len(args) < 2 {
		args = append(args, "")
	}
	if args[1] == "--help" {
		fmt.Println(" --login :- Add a gmail account (More than one account could be added this way)\n --help :- Show help\n --list :- Show all emails")
		return
	}

	helpers.Create_folder(paths.CONFIG_FOLDER)

	//This is a temporary function to copy assets. Should be removed when assets folders are created by the installation
	helpers.Copy_asset(paths.ASSETS_SOURCE_PATH, paths.ASSETS_PATH)
	ctx := context.Background()
	config_json, err := os.ReadFile(paths.CREDENTIALS_FILE)
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(config_json, gmail.GmailReadonlyScope)
	if err != nil {
		fmt.Printf("Unable to read client secret file: %v\n Follow the steps 'Enable the API' and 'Authorize credentials for a desktop application' from the following page\n https://developers.google.com/gmail/api/quickstart/go \n Note:- Ignore all other steps\n rename the downloaded file to credentials.json and copy it to\n~/.config/gmail_watcher", err)
		panic("No client secret file")
	}
	err = gmail_client.Change_server_port(config, paths.PORT)

	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	var tokFiles []string

	tokFiles, err = helpers.Load_json_list(paths.LOGIN_TOKENS_LIST_FILE)
	if err != nil {
		tokFiles = make([]string, 0)
	}

	if args[1] == "--login" || len(tokFiles) == 0 {
		if len(tokFiles) == 0 && args[1] != "--login" {
			// fmt.Println("No saved accounts found Log in")
			log.Println("No saved accounts found Log in")
		}
		token := gmail_client.GetTokenFromWeb(config)
		token_file_path, err := gmail_client.Add_token(&tokFiles)
		if err != nil {
			fmt.Printf("Error Occured")
			log.Printf("Error adding token :- %v", err)
		}
		helpers.SaveToken(*token_file_path, token)
	}

	var max_retries uint8 = 3
	var client_srvs []*gmail_client.ClientService
	client_srvs, shouldReturn := daemon.GetClientSrvs(ctx, max_retries, client_srvs, err, config, tokFiles)
	if shouldReturn {
		return
	}
	if args[1] == "--list" {
		list_len := 15
		for _, client_srv := range client_srvs {
			var wg sync.WaitGroup
			mailMessage := make(chan string)
			//log.Println("Serving", client_srv)
			// msgs, err := client_srv.Update(false)
			msgs, err := client_srv.GetMsgIDs()
			if err != nil {
				fmt.Println("Error getting emails msg ids")
				log.Fatalf("Error getting emails msg ids:- %v", err)
				return
			} else {
				fmt.Printf("Email:- %s\n", client_srv.EmailID)
				for index, msg := range msgs.Messages[:list_len] {
					wg.Add(1)
					go func(client_srv *gmail_client.ClientService, msg *gmail.Message, index int) {
						msg_mail, err := client_srv.GetMsg("me", msg.Id)
						if err != nil {
							log.Printf("Error getting emails:- %v", err)
							fmt.Print("Error getting emails")
						} else {
							mailMessage <- msg_mail.Snippet
						}
						defer wg.Done()
						return
					}(client_srv, msg, index)

				}

			}

			// Launch a goroutine to close the channel after sending is done
			go func() {
				wg.Wait()                // Wait for all senders to finish
				defer close(mailMessage) // Close the channel after all sends are complete
			}()
			for msg_c := range mailMessage {
				fmt.Printf("%s\n", msg_c)
			}
			fmt.Println("-----")
		}

		return
	}
	if args[1] == "--daemon" {
		daemon.Run(3, client_srvs)
	}
}
