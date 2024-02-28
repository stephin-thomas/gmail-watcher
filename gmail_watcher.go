package main

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

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

type clientService struct {
	gmail_service *gmail.Service
	email_id      string
	db            string
	id_db         *map[string]struct{}
}

var CONFIG_FOLDER string = get_config_folder()

var CREDENTIALS_FILE = filepath.Join(CONFIG_FOLDER, "credentials.json")
var LOGIN_TOKENS_LIST_FILE = filepath.Join(CONFIG_FOLDER, "login_tokens.json")
var PORT int64 = 5000
var NOTIFICATION_ICON = filepath.Join(CONFIG_FOLDER, "assets/email_notify.webp")

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	args := os.Args
	if len(args) < 2 {
		args = append(args, "")
	}
	if args[1] == "--help" {
		fmt.Println(" --login :- Add a gmail account (More than one account could be added this way)\n --help :- Show help")
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

	if args[1] == "--login" {
		token := getTokenFromWeb(config)
		token_file_path := add_token(&tokFiles)
		saveToken(*token_file_path, token)
	}
	if len(tokFiles) == 0 {
		fmt.Println("No saved accounts found try again")
		return
	}
	max_retries := 3
	var client_srvs []*clientService
	for i := 0; i < max_retries; i++ {
		client_srvs = collect_gmail_serv(config, &ctx, &tokFiles, &CONFIG_FOLDER)
		if !(len(client_srvs) == 0) {
			break
		}
	}
	if len(client_srvs) == 0 {
		fmt.Println("Couldn't construct any clients")
		return
	}
	for {
		retry := 0
		for _, client_srv := range client_srvs {
			//log.Println("Serving", client_srv)
			err := email_main(client_srv)
			if err != nil {
				retry = retry + 1
				log.Println("Sleeping:-", 10*time.Second)
				beeep.Notify("Error", "Unable retrieve emails please check your internet connection", NOTIFICATION_ICON)
				time.Sleep(10 * time.Second)
				if retry == max_retries {
					beeep.Notify("Error", "Shutting down Gmail Watcher due to errors", NOTIFICATION_ICON)
					return
				}
			}
		}
		log.Println("Sleeping:-", 30*time.Second)
		time.Sleep(30 * time.Second)
	}
}

func SlicePop[T any](s []T, i int) []T {
	s = append(s[:i], s[i+1:]...)
	return s
}

func refresh_token_if_expired(CONFIG_FOLDER string, tokFiles []string) []string {
	var newtokFiles []string
	newtokFiles = tokFiles
	// copy(newtokFiles, tokFiles)
	for i, tokFile := range tokFiles {
		log.Println("Checking if token is expired:-", tokFile)
		tok, err := tokenFromFile(tokFile)
		if err != nil {
			log.Println("Error creating token from token file:-", err)
		} else {
			expiry_time := tok.Expiry
			if token_expired(&expiry_time) {
				beeep.Notify("Token Expired", fmt.Sprintln("Token expired on", &expiry_time, "Relogin to continue using Gmail Notifier"), NOTIFICATION_ICON)
				log.Println("Removing expired token:-", tokFile[i])
				newtokFiles = SlicePop(newtokFiles, i)
				add_token(&newtokFiles)
			} else {
				log.Println("Found token not expired:-", tokFile[i])

			}
		}
	}
	log.Printf("Initial Token Files = %v\n Final Token Files after removing errors %v", tokFiles, newtokFiles)
	return newtokFiles
}

func collect_gmail_serv(config *oauth2.Config, ctx *context.Context, tokFiles *[]string, CONFIG_FOLDER *string) []*clientService {
	log.Println("Collecting Gmail Clients from configuration from tokens", tokFiles)
	var gmail_services []*clientService

	for _, tokFile := range *tokFiles {
		client := getClient(config, tokFile)

		srv, err := get_gmail_serv(client, ctx)
		// db_file := fmt.Sprintf("id_db_%s", tokFile)
		// db := path.Join(*CONFIG_FOLDER, db_file)
		// log.Println("DB created at ", db)
		email, err := get_email(srv)
		db_file := fmt.Sprintf("id_db_%s.json", email.EmailAddress)
		db := path.Join(*CONFIG_FOLDER, db_file)
		log.Println("Using DB at", db)
		for err != nil {
			srv, err = get_gmail_serv(client, ctx)
			log.Println("Couldn't create a gmail service trying again in 30s")
			time.Sleep(30 * time.Second)
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
	return gmail_services
}

func get_email(gmail_service *gmail.Service) (*gmail.Profile, error) {
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

func email_main(client_srv *clientService) error {
	user := "me"
	msg_list, err := get_msg_ids(client_srv.gmail_service, user)
	if err != nil {
		return err
	}
	//var updated_emails []string
	log.Printf("Total msgs from google:- %d\n Using only:- 15", len(msg_list.Messages))
	msgs := msg_list.Messages[0:15]
	id_db_update := false
	for _, msg := range msgs {
		if !check_if_value_present(msg.Id, client_srv.id_db) {
			msg, err := get_msg(client_srv.gmail_service, user, msg.Id)
			(*client_srv.id_db)[msg.Id] = struct{}{}
			id_db_update = true
			if err == nil {
				// profile, _ := get_email(client_srv.gmail_service)
				// email_id := profile.EmailAddress
				// log.Println(*msg)
				show_emails(msg, &client_srv.email_id)
			} else {
				log.Fatalf("error occured getting email %v", err)
			}
		}
	}
	if id_db_update {
		err := client_srv.save()
		if err != nil {
			log.Fatalln("Error saving db database", client_srv.db, err)
		}
	}
	return nil
}

func show_emails(msg *gmail.Message, user_email *string) {
	err := beeep.Notify(fmt.Sprintf("Gmail Watcher:-%s", *user_email), msg.Snippet, NOTIFICATION_ICON)
	if err != nil {
		log.Println("Error during notification", err)
	}
	//log.Println(msg)
	//log.Println(msg.Snippet)
}

// func get_updated_emails(msg_list *[]*gmail.Message, client_srv *clientService) *map[string]struct{} {
// 	id_list := create_id_list(msg_list)
// 	diff := list_difference(id_list, client_srv.id_db)
// 	if len(*diff) > 0 {
// 		fmt.Println("Found Difference")
// 		serialize_n_save(id_list, client_srv.db)
// 		client_srv.id_db = id_list
// 	} else {
// 		log.Println("No new emails found")

// 	}
// 	return diff
// }

func get_msg_ids(srv *gmail.Service, user string) (*gmail.ListMessagesResponse, error) {
	msg_list, err := srv.Users.Messages.List(user).Do()
	return msg_list, err
}

func get_msg(srv *gmail.Service, user string, msg_id string) (*gmail.Message, error) {
	msg, err := srv.Users.Messages.Get(user, msg_id).Do()
	return msg, err

}
