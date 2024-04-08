package daemon

import (
	"context"
	"strings"

	"log"
	"net/http"
	"time"

	"github.com/coreos/go-systemd/daemon"

	"github.com/gen2brain/beeep"
	"github.com/gmail-watcher/gmail_client"
	"github.com/gmail-watcher/helpers"
	"github.com/gmail-watcher/paths"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func Run(max_retries uint8, client_srvs []*gmail_client.ClientService) {
	daemon.SdNotify(true, daemon.SdNotifyReady)
	shutdown := false
	for !shutdown {
		var retry uint8 = 0
		for _, client_srv := range client_srvs {
			//log.Println("Serving", client_srv)
			_, err := client_srv.Run(true)
			if err != nil {
				retry = retry + 1
				log.Println("Sleeping:- 10 sec")
				beeep.Notify("Error", "Unable retrieve emails please check your internet connection", paths.NOTIFICATION_ICON)
				time.Sleep(10 * time.Second)
				if retry == max_retries {
					beeep.Notify("Error", "Shutting down Gmail Watcher due to errors", paths.NOTIFICATION_ICON)
					shutdown = true
					break
				}
			} else {
				retry = 0
			}
		}
		log.Println("Sleeping:-", 30*time.Second)
		time.Sleep(30 * time.Second)
	}

	daemon.SdNotify(true, daemon.SdNotifyStopping)
}

func Handle_client_srvs(ctx context.Context, max_retries uint8, client_srvs []*gmail_client.ClientService, err error, config *oauth2.Config, tokFiles []string) ([]*gmail_client.ClientService, bool) {
	var i uint8
	for i = 0; i < max_retries; i++ {
		client_srvs, err = collect_gmail_serv(config, &ctx, &tokFiles, &paths.CONFIG_FOLDER)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Println("Couldn't construct any clients")
		beeep.Notify("Fatal Error", "Couldn't construct any clients exiting", paths.NOTIFICATION_ICON)
		return nil, true
	}
	if len(client_srvs) == 0 {
		log.Println("No clients found")
		beeep.Notify("Fatal Error", "No clients found", paths.NOTIFICATION_ICON)
		return nil, true
	}
	return client_srvs, false
}

func collect_gmail_serv(config *oauth2.Config, ctx *context.Context, tokFiles *[]string, CONFIG_FOLDER *string) ([]*gmail_client.ClientService, error) {
	log.Println("Collecting Gmail Clients from configuration from tokens", tokFiles)
	var gmail_services []*gmail_client.ClientService

	for _, tokFile := range *tokFiles {
		client := gmail_client.GetClient(config, tokFile)
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
		id_db, err := helpers.Load_old_msg_ids(db)
		if err != nil {
			id_db = make(map[string]struct{})
		}
		client_service := gmail_client.ClientService{
			GmailService: srv,
			ID_DB:        &id_db,
			DB:           db,
			EmailID:      email.EmailAddress,
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
