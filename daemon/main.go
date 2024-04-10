package daemon

import (
	"context"

	"log"
	"time"

	"github.com/coreos/go-systemd/daemon"

	"github.com/gen2brain/beeep"
	"github.com/gmail-watcher/gmail_client"
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
			err := client_srv.Update(true)
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
