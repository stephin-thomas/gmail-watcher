package main

import (
	"context"
	"fmt"
	"sync"

	"path/filepath"
	"strings"

	"github.com/gen2brain/beeep"

	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

var CONFIG_FOLDER string = get_config_folder()
var CREDENTIALS_FILE = filepath.Join(CONFIG_FOLDER, "credentials.json")
var LOGIN_TOKENS_LIST_FILE = filepath.Join(CONFIG_FOLDER, "login_tokens.json")
var PORT int64 = 5000
var NOTIFICATION_ICON = filepath.Join(CONFIG_FOLDER, "assets/notification.png")

func main() {
	log_file_path := filepath.Join(CONFIG_FOLDER, "gmail-watcher.log")
	logFile, err := os.OpenFile(log_file_path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Redirect log output to the file
	log.SetOutput(logFile)
	log.Println("Config Folder:-", CONFIG_FOLDER)
	args := os.Args
	if len(args) < 2 {
		args = append(args, "")
	}
	if args[1] == "--help" {
		fmt.Println(" --login :- Add a gmail account (More than one account could be added this way)\n --help :- Show help\n --list :- Show all emails")
		return
	}

	create_folder(CONFIG_FOLDER)
	assets_source_path := "assets/notification.png"
	assets_path := filepath.Join(CONFIG_FOLDER, assets_source_path)

	//This is a temporary function to copy assets. Should be removed when assets folders are created by the installation
	copy_asset(assets_source_path, assets_path)
	ctx := context.Background()
	config_json, err := os.ReadFile(CREDENTIALS_FILE)
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(config_json, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v\n Follow the steps 'Enable the API' and 'Authorize credentials for a desktop application' from the following page\n https://developers.google.com/gmail/api/quickstart/go \n Note:- Ignore all other steps\n rename the downloaded file to credentials.json and copy it to\n~/.config/gmail_watcher", err)
	}
	change_server_port(config, PORT)

	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	var tokFiles []string

	tokFiles, err = load_existing_tokens()
	if err != nil {
		tokFiles = make([]string, 0)
	}

	if args[1] == "--login" || len(tokFiles) == 0 {
		if len(tokFiles) == 0 && args[1] != "--login" {
			fmt.Println("No saved accounts found Log in")
		}
		token := getTokenFromWeb(config)
		token_file_path := add_token(&tokFiles)
		saveToken(*token_file_path, token)
	}

	max_retries := 3
	var client_srvs []*clientService
	client_srvs, shouldReturn := handle_client_srvs(ctx, max_retries, client_srvs, err, config, tokFiles)
	if shouldReturn {
		return
	}
	if args[1] == "--list" {
		list_len := 15
		for _, client_srv := range client_srvs {
			var wg sync.WaitGroup
			mailMessage := make(chan string)
			//log.Println("Serving", client_srv)
			msgs, err := client_srv.run(false)
			if err != nil {
				println("Error getting emails")
				return
			} else {
				fmt.Printf("Email:- %s\n", client_srv.email_id)
				for index, msg := range msgs[:list_len] {
					wg.Add(1)
					go func(client_srv *clientService, msg *gmail.Message, index int) {
						msg_mail, err := get_msg(client_srv.gmail_service, "me", msg.Id)
						if err != nil {
							fmt.Println("Error getting emails")
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
	for {
		retry := 0
		for _, client_srv := range client_srvs {
			//log.Println("Serving", client_srv)
			_, err := client_srv.run(true)
			if err != nil {
				retry = retry + 1
				log.Println("Sleeping:- 10 sec")
				beeep.Notify("Error", "Unable retrieve emails please check your internet connection", NOTIFICATION_ICON)
				time.Sleep(10 * time.Second)
				if retry == max_retries {
					beeep.Notify("Error", "Shutting down Gmail Watcher due to errors", NOTIFICATION_ICON)
					return
				}
			} else {
				retry = 0
			}
		}
		log.Println("Sleeping:-", 30*time.Second)
		time.Sleep(30 * time.Second)
	}
}

func handle_client_srvs(ctx context.Context, max_retries int, client_srvs []*clientService, err error, config *oauth2.Config, tokFiles []string) ([]*clientService, bool) {
	for i := 0; i < max_retries; i++ {
		client_srvs, err = collect_gmail_serv(config, &ctx, &tokFiles, &CONFIG_FOLDER)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Println("Couldn't construct any clients")
		beeep.Notify("Fatal Error", "Couldn't construct any clients exiting", NOTIFICATION_ICON)
		return nil, true
	}
	if len(client_srvs) == 0 {
		log.Println("No clients found")
		beeep.Notify("Fatal Error", "No clients found", NOTIFICATION_ICON)
		return nil, true
	}
	return client_srvs, false
}

func collect_gmail_serv(config *oauth2.Config, ctx *context.Context, tokFiles *[]string, CONFIG_FOLDER *string) ([]*clientService, error) {
	log.Println("Collecting Gmail Clients from configuration from tokens", tokFiles)
	var gmail_services []*clientService

	for _, tokFile := range *tokFiles {
		client := getClient(config, tokFile)
		srv, err := get_gmail_serv(client, ctx)
		email, err := get_email_prof(srv)

		db := strings.Replace(tokFile, "token_", "id_db_", -1)
		// db = fmt.Sprintf("id_db_%s")
		// db_file := fmt.Sprintf("id_db_%s.json", email.EmailAddress)
		// db := path.Join(*CONFIG_FOLDER, db_file)
		log.Println("Using DB at", db)
		for err != nil {
			return nil, err
		}
		id_db, err := load_old_msg_ids(db)
		if err != nil {
			id_db = make(map[string]struct{})
		}
		client_service := clientService{
			gmail_service: srv,
			id_db:         &id_db,
			db:            db,
			email_id:      email.EmailAddress,
		}
		log.Println("Successfully created client")
		gmail_services = append(gmail_services, &client_service)
		//get_email(&client_service)
	}
	return gmail_services, nil
}

func get_email_prof(gmail_service *gmail.Service) (*gmail.Profile, error) {
	usr_name, err := gmail_service.Users.GetProfile("me").Do()

	if err != nil {
		log.Fatal("Error getting email profile")

	}
	return usr_name, err
}

func get_gmail_serv(client *http.Client, ctx *context.Context) (*gmail.Service, error) {
	srv, err := gmail.NewService(*ctx, option.WithHTTPClient(client))
	return srv, err

}
