package daemon

import (
	"context"

	"log"
	"time"

	"github.com/coreos/go-systemd/daemon"

	"github.com/gen2brain/beeep"
	"github.com/gmail-watcher/gmail_client"
	"github.com/gmail-watcher/helpers"
	"github.com/gmail-watcher/paths"
	"golang.org/x/oauth2"
)

func Run(max_retries uint8, client_srvs []*gmail_client.ClientService) {
	daemon.SdNotify(true, daemon.SdNotifyReady)
	shutdown := false
	for !shutdown {
		var retry uint8 = 0
		for _, client_srv := range client_srvs {
			//log.Println("Serving", client_srv)
			// err := client_srv.Update(true)
			user := "me"
			updated_msg_list, err1 := client_srv.UpdateMsgIDs()
			//var updated_emails []string
			log.Printf("Total msgs from google:- %d\n Using only:- 15", len(updated_msg_list))
			// msgs := msg_list.Messages[0:15]
			var max_shown int8 = 15
			var shown_index int8 = 0
			var err2 error
			for _, msg_id := range updated_msg_list {
				// if !client_srv.find_msg(*msg_id) {
				shown_index += 1
				msg, err2 := client_srv.GetMsg(user, *msg_id)
				if err2 != nil {
					log.Printf("Error getting email with id %v:- %v", *msg_id, err2)
				}
				if max_shown > shown_index {
					helpers.Notify(&msg.Snippet, &client_srv.EmailID)
				}
				// }
			}

			if err1 != nil || err2 != nil {
				retry = retry + 1
				log.Println("Sleeping:- 10 sec")
				if retry == 1 {
					err_msg := "Unable retrieve emails please check your internet connection"
					err_title := "Error"
					helpers.Notify(&err_msg, &err_title)
				}
				time.Sleep(10 * time.Second)
				if retry == max_retries {
					err_msg := "Shutting down Gmail Watcher due to errors"
					err_title := "Error"
					helpers.Notify(&err_msg, &err_title)
					shutdown = true
				}
				break
			} else {
				retry = 0
			}
			if len(updated_msg_list) > 0 {
				err := client_srv.Save()
				if err != nil {
					log.Fatalln("Error saving db database", client_srv.DB_Path, err)
				}
			}

		}
		log.Println("Sleeping:-", 30*time.Second)
		time.Sleep(30 * time.Second)
	}

	daemon.SdNotify(true, daemon.SdNotifyStopping)
}

func GetClientSrvs(ctx context.Context, max_retries uint8, client_srvs []*gmail_client.ClientService, err error, config *oauth2.Config, tokFiles []string) ([]*gmail_client.ClientService, bool) {
	var i uint8
	for i = 0; i < max_retries; i++ {
		client_srvs, err = gmail_client.CollectGmailServ(config, &ctx, &tokFiles, &paths.CONFIG_FOLDER)
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
