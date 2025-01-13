package daemon

import (
	"fmt"
	"log"
	"time"

	"github.com/coreos/go-systemd/daemon"
	"github.com/gmail-watcher/gmail_client"

	"github.com/gmail-watcher/io_helpers"
)

func RunDaemon(max_retries uint8, max_notifications uint8, client_srvs *[]*gmail_client.GmailService) error {
	log.Println("Daemon started")
	err := io_helpers.Notify("Started", "Gmail Watcher")
	if err != nil {
		log.Fatalln("Error occured notification not working.", err)
	}
	_, _ = daemon.SdNotify(true, daemon.SdNotifyReady)
	shutdown := false
	for !shutdown {
		var retry uint8 = 0
		i := 0
		for i < len(*client_srvs) {
			client_srv := (*client_srvs)[i]
			updated_msg_list, err1 := client_srv.UpdateMsgIDs()
			if err1 != nil {
				retry = retry + 1
				log.Println("Error getting msg list sleeping 10 sec and trying again", err1)
				time.Sleep(10 * time.Second)
				continue
			}
			handle_retries(&retry, max_retries, &shutdown)

			if len(updated_msg_list) > 0 {
				err := client_srv.Save()
				if err != nil {
					_ = io_helpers.Notify("Error saving database", "Error!")
					log.Printf("error saving db database %v\n %v", client_srv.DB_Path, err)
					return fmt.Errorf("error saving db database %v\n %w", client_srv.DB_Path, err)
				}
				log.Printf("Total msgs from google:- %d\n Using only:- 8", len(updated_msg_list))
				if len(updated_msg_list) > 8 {
					updated_msg_list = updated_msg_list[:8]
				}
				msgs, err2 := client_srv.FetchAllMail(updated_msg_list)

				if err2 != nil {
					retry = retry + 1
					time.Sleep(10 * time.Second)
					log.Println("error getting messages for the updated msg ids. sleeping 10 secs and trying again", err2)
					continue
				} else {
					retry = 0
					for _, msg := range *msgs {
						sender := gmail_client.GetSender(msg)
						_ = io_helpers.Notify(msg.Snippet, sender)
						// time.Sleep(3 * time.Second)
					}
				}

			}
			i++
		}
		log.Println("Sleeping:-", 30)
		if retry == max_retries {
			_ = io_helpers.Notify("Unable to retrieve email", "Gmail Watcher Error!")
		}
		time.Sleep(30 * time.Second)
	}

	_, _ = daemon.SdNotify(true, daemon.SdNotifyStopping)
	return nil
}

func handle_retries(retry *uint8, max_retries uint8, shutdown *bool) {
	if *retry == 1 {
		err_msg := "Unable retrieve emails please check your internet connection"
		err_title := "Error"
		_ = io_helpers.Notify(err_msg, err_title)
	}
	time.Sleep(10 * time.Second)
	if *retry == max_retries {
		err_msg := "Shutting down Gmail Watcher due to errors"
		err_title := "Error"
		_ = io_helpers.Notify(err_msg, err_title)
		*shutdown = true
	}
}
