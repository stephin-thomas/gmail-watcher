package main

import (
	"context"
	//"encoding/json"
	"fmt"
	"github.com/gen2brain/beeep"
	//"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
	// "io"
	// b64 "encoding/base64"
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

func main() {
	args := os.Args
	if !(len(args) > 1) {
		args = append(args, "None")
	}
	if args[1] == "--help" {
		fmt.Println(" login :- Add a gmail account (More than one account could be added this way)\n --help :- Show help")
		return
	}
	var CONFIG_FOLDER string = get_config_folder()
	var web_server WebServer = WebServer{
		server_running: false,
		channel_main:   make(chan string, 10),
		wg:             &sync.WaitGroup{},
	}
	create_config_folder(CONFIG_FOLDER)
	ctx := context.Background()
	change_port_creds(&CONFIG_FOLDER)
	b, err := os.ReadFile(CONFIG_FOLDER + "credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v\n Follow the steps 'Enable the API' and 'Authorize credentials for a desktop application' from the following page\n https://developers.google.com/gmail/api/quickstart/go \n Note:- Ignore all other steps\n rename the downloaded file to credentials.json and copy it to\n~/.config/gmail_watcher", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	tokFiles, err := load_old_ids(CONFIG_FOLDER + "Config.json")
	if err != nil || args[1] == "login" {
		if err != nil {
			fmt.Println("Error loading config.json")
		}
		tokFiles = gen_random_token_name(tokFiles, &CONFIG_FOLDER)
		save_as_json(tokFiles, CONFIG_FOLDER+"Config.json")
	}
	client_srvs := collect_gmail_serv(config, &ctx, &tokFiles, &web_server, &CONFIG_FOLDER)
	for {
		for _, client_srv := range client_srvs {
			err := email_main(client_srv)
			if err != nil {
				time.Sleep(10 * time.Second)
			}
		}
		time.Sleep(30 * time.Second)
	}
}
func collect_gmail_serv(config *oauth2.Config, ctx *context.Context, tokFiles *[]string, web_server *WebServer, CONFIG_FOLDER *string) []*clientService {
	var gmail_services []*clientService
	for i, tokFile := range *tokFiles {
		db := fmt.Sprintf("%sid_db%d.json", *CONFIG_FOLDER, i)
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
		usr_name, err := client_service.gmail_service.Users.GetProfile("me").Do()
		if err == nil {
			fmt.Println("Created", usr_name.EmailAddress)
		}

	}
	return gmail_services
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
	var updated_emails []string
	updated_emails = get_updated_emails(msg_list, client_srv)
	for i, msg := range updated_emails {
		if i > 5 {
			break
		}
		msg, err := get_msg(client_srv.gmail_service, user, msg)
		if err == nil {
			show_emails(msg)
		}
	}
	return nil
}

func show_emails(msg *gmail.Message) {
	err := beeep.Notify("New Email Received", msg.Snippet, "assets/email_notify.webp")
	if err != nil {
		log.Println("Error during notification", err)
	}
	log.Println(msg.Snippet)
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
