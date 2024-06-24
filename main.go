package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/gmail-watcher/common"
	"github.com/gmail-watcher/gmail_client"
	"github.com/gmail-watcher/io_helpers"
	"github.com/gmail-watcher/paths"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

func init() {
	logFile, err := os.OpenFile(paths.LOG_FILE_PATH, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Redirect log output to the file
	log.SetOutput(logFile)
	log.Println("Config Folder:-", paths.CONFIG_FOLDER)
}

func main() {
	cli_ctx := kong.Parse(&CLI)
	io_helpers.Create_folder(paths.CONFIG_FOLDER)

	//This is a temporary function to copy assets. Should be removed when assets folders are created by the installation
	io_helpers.Copy_asset(paths.ASSETS_SOURCE_PATH, paths.ASSETS_PATH)
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

	tokFiles, err = io_helpers.Load_json_list(paths.LOGIN_TOKENS_LIST_FILE)
	if err != nil {
		tokFiles = make([]string, 0)
	}
	// var max_retries uint8 = 3
	max_retries := CLI.Gmail.MaxRetries
	switch cli_ctx.Command() {
	case "login":
		{
			token := common.GetTokenFromWeb(config)
			token_file_path, err := gmail_client.Add_token(&tokFiles)
			if err != nil {
				fmt.Printf("Error Occured")
				log.Printf("Error adding token :- %v", err)
			}
			io_helpers.SaveToken(*token_file_path, token)
		}

	case "gmail list":
		{
			clientSrvs, loginNotFound := gmail_client.GetClientSrvs(ctx, max_retries, config, tokFiles)
			if loginNotFound {
				log.Fatalf("No Logins found \n%s", tokFiles)
				return
			}
			list_len := CLI.Gmail.List.ListLen
			// list_len := 15
			for _, client_srv := range clientSrvs {
				var wg sync.WaitGroup
				mailMessage := make(chan string)
				msgs, err := client_srv.GetMsgIDs()
				if err != nil {
					fmt.Println("Error getting emails msg ids")
					log.Fatalf("Error getting emails msg ids:- %v", err)
					return
				} else {
					fmt.Printf("Email:- %s\n", client_srv.EmailID)
					for index, msg := range msgs.Messages[:list_len] {
						wg.Add(1)
						go func(client_srv *gmail_client.GmailService, msg *gmail.Message, index int) {
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
	case "gmail daemon":
		{
			clientSrvs, loginNotFound := gmail_client.GetClientSrvs(ctx, max_retries, config, tokFiles)
			if loginNotFound {
				log.Fatalf("Login not found when running daemon")
			}
			gmail_client.RunDaemon(3, clientSrvs)
		}
	default:
		panic(cli_ctx.Command())
	}
}
