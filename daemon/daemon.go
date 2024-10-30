package daemon

import (
	"log"
	"time"

	"github.com/coreos/go-systemd/daemon"
	"github.com/gmail-watcher/gmail_client"

	"github.com/gmail-watcher/io_helpers"
)

func RunDaemon(max_retries uint8, max_notifications uint8, client_srvs []*gmail_client.GmailService) {
	daemon.SdNotify(true, daemon.SdNotifyReady)
	shutdown := false
	for !shutdown {
		var retry uint8 = 0
		for _, client_srv := range client_srvs {
			updated_msg_list, err1 := client_srv.UpdateMsgIDs()
			if err1 != nil {
				retry = retry + 1
				log.Println("Sleeping:- 10 sec")
				break
			}
			handle_retries(&retry, max_retries, &shutdown)

			if len(updated_msg_list) > 0 {
				err := client_srv.Save()
				if err != nil {
					log.Fatalln("Error saving db database", client_srv.DB_Path, err)
				}
				log.Printf("Total msgs from google:- %d\n Using only:- 15", len(updated_msg_list))
				msgs, err2 := gmail_client.FetchMail(client_srv, updated_msg_list)

				if err2 != nil {
					retry = retry + 1
					log.Println("Sleeping:- 10 sec")
					break
				} else {
					retry = 0
					for _, msg := range *msgs {
						sender := gmail_client.GetSender(msg)
						io_helpers.Notify(&msg.Snippet, &sender)
						time.Sleep(3 * time.Second)
					}
				}

			}
		}
		log.Println("Sleeping:-", 30)
		time.Sleep(30 * time.Second)
	}

	daemon.SdNotify(true, daemon.SdNotifyStopping)
}

func handle_retries(retry *uint8, max_retries uint8, shutdown *bool) {
	if *retry == 1 {
		err_msg := "Unable retrieve emails please check your internet connection"
		err_title := "Error"
		io_helpers.Notify(&err_msg, &err_title)
	}
	time.Sleep(10 * time.Second)
	if *retry == max_retries {
		err_msg := "Shutting down Gmail Watcher due to errors"
		err_title := "Error"
		io_helpers.Notify(&err_msg, &err_title)
		*shutdown = true
	}
}
