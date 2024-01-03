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
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type WebServer struct {
	server_running bool
	web_server     *http.Server
	channel_main   chan string
	wg             *sync.WaitGroup
}

type clientService struct {
	gmail_service *gmail.Service
	db            string
	old_ids       []string
}

var CONFIG_FOLDER string = get_config_folder()

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	args := os.Args
	if !(len(args) > 1) {
		args = append(args, "None")
	}
	if args[1] == "--help" {
		fmt.Println(" --login :- Add a gmail account (More than one account could be added this way)\n --help :- Show help")
		return
	}

	var web_server WebServer = WebServer{
		server_running: false,
		channel_main:   make(chan string, 10),
		wg:             &sync.WaitGroup{},
	}
	create_folder(CONFIG_FOLDER)
	assets_source_path := "assets/notification.png"
	assets_path := filepath.Join(CONFIG_FOLDER, assets_source_path)
	//This is a temporary function to copy assets. Should be removed when assets folders are created by the installation
	copy_asset(assets_source_path, assets_path)
	ctx := context.Background()
	change_server_port(&CONFIG_FOLDER, 5000)
	CREDENTIALS_FILE, err := os.ReadFile(filepath.Join(CONFIG_FOLDER, "credentials.json"))
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v\n Follow the steps 'Enable the API' and 'Authorize credentials for a desktop application' from the following page\n https://developers.google.com/gmail/api/quickstart/go \n Note:- Ignore all other steps\n rename the downloaded file to credentials.json and copy it to\n~/.config/gmail_watcher", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(CREDENTIALS_FILE, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	var tokFiles []string
	if args[1] == "--login" {
		tokFiles = load_token_files(CONFIG_FOLDER, true)
	} else {
		tokFiles = load_token_files(CONFIG_FOLDER, false)
	}
	//tokFiles = refresh_token_if_expired(CONFIG_FOLDER, tokFiles)
	client_srvs := collect_gmail_serv(config, &ctx, &tokFiles, &web_server, &CONFIG_FOLDER)
	for {
		if len(client_srvs) == 0 {
			client_srvs = collect_gmail_serv(config, &ctx, &tokFiles, &web_server, &CONFIG_FOLDER)
		}
		for _, client_srv := range client_srvs {
			//log.Println("Serving", client_srv)
			err := email_main(client_srv)
			if err != nil {
				log.Println("Sleeping:-", 10*time.Second)
				time.Sleep(10 * time.Second)
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
				beeep.Notify("Token Expired", fmt.Sprintln("Token expired on", &expiry_time, "Relogin to continue using Gmail Notifier"), "assets/email_notify.webp")
				log.Println("Removing expired token:-", tokFile[i])
				newtokFiles = SlicePop(newtokFiles, i)
				newtokFiles = add_token(newtokFiles, CONFIG_FOLDER)
			} else {
				log.Println("Found token not expired:-", tokFile[i])

			}
		}
	}
	log.Printf("Initial Token Files = %v\n Final Token Files after removing errors %v", tokFiles, newtokFiles)
	return newtokFiles
}

func collect_gmail_serv(config *oauth2.Config, ctx *context.Context, tokFiles *[]string, web_server *WebServer, CONFIG_FOLDER *string) []*clientService {
	log.Println("Collecting Gmail Clients from configuration from tokens", tokFiles)
	var gmail_services []*clientService

	for i, tokFile := range *tokFiles {
		db_file := fmt.Sprintf("id_db%d.json", i)
		db := path.Join(*CONFIG_FOLDER, db_file)
		log.Println("DB created at ", db)
		//db := fmt.Sprintf("%sid_db%d.json", *CONFIG_FOLDER, i)
		client := getClient(config, tokFile, web_server)

		srv, err := get_gmail_serv(client, ctx)
		for err != nil {
			srv, err = get_gmail_serv(client, ctx)
			log.Println("Couldn't create a gmail service trying again in 30s")
			time.Sleep(30 * time.Second)
		}

		client_service := clientService{
			gmail_service: srv,
			db:            db,
			old_ids:       []string{},
		}
		log.Println("Successfully created client")
		gmail_services = append(gmail_services, &client_service)
		get_email(&client_service)
	}
	return gmail_services
}

func get_email(client_service *clientService) (*gmail.Profile, error) {
	usr_name, err := client_service.gmail_service.Users.GetProfile("me").Do()

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
	var updated_emails []string = get_updated_emails(msg_list, client_srv)
	for i, msg := range updated_emails {
		if i > 5 {
			break
		}
		msg, err := get_msg(client_srv.gmail_service, user, msg)
		if err == nil {
			profile, _ := get_email(client_srv)
			email_id := profile.EmailAddress
			show_emails(msg, &email_id)
		}
	}
	return nil
}

func show_emails(msg *gmail.Message, user_email *string) {
	err := beeep.Notify(fmt.Sprintf("Email:-%s", *user_email), msg.Snippet, filepath.Join(CONFIG_FOLDER, "assets/notification.png"))
	if err != nil {
		log.Println("Error during notification", err)
	}
	//log.Println(msg)
	//log.Println(msg.Snippet)
}
func get_updated_emails(msg_list *gmail.ListMessagesResponse, client_srv *clientService) []string {
	if len(client_srv.old_ids) == 0 {
		old_ids, err := load_old_ids(client_srv.db)
		if err == nil {
			client_srv.old_ids = old_ids
		}
		// 	start = false
	}

	id_list := create_id_list(msg_list.Messages)
	diff := list_difference(id_list, client_srv.old_ids)
	if len(diff) > 0 {
		save_as_json(id_list, client_srv.db)
		client_srv.old_ids = id_list
	}
	return diff

}
func get_msg_ids(srv *gmail.Service, user string) (*gmail.ListMessagesResponse, error) {
	msg_list, err := srv.Users.Messages.List(user).Do()
	return msg_list, err
}

func get_msg(srv *gmail.Service, user string, msg_id string) (*gmail.Message, error) {
	msg, err := srv.Users.Messages.Get(user, msg_id).Do()
	return msg, err

}
