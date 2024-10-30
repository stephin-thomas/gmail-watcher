package gmail_client

import (
	"context"

	"log"

	"github.com/gen2brain/beeep"
	"github.com/gmail-watcher/common"
	"github.com/gmail-watcher/paths"
)

func GetClientSrvs(clients []*common.LocalClient, ctx context.Context, max_retries uint8) ([]*GmailService, bool) {
	var i uint8
	var client_srvs []*GmailService
	var err error
	for i = 0; i < max_retries; i++ {
		client_srvs, err = CollectGmailServ(clients, &ctx, &paths.CONFIG_FOLDER)
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
