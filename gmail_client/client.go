package gmail_client

import (
	"strings"

	"context"
	"log"

	"github.com/gmail-watcher/common"
	"github.com/gmail-watcher/io_helpers"
	"google.golang.org/api/gmail/v1"
)

type GmailService struct {
	GmailService *gmail.Service
	EmailID      string
	DB_Path      string
	ID_DB        map[string]struct{}
}

func (c *GmailService) Save() error {
	err := io_helpers.SerializeNsave(c.ID_DB, c.DB_Path)
	return err
}
func (c *GmailService) find_msg(needle string) bool {
	_, found := (c.ID_DB)[needle]
	return found
}

func (c *GmailService) GetMsgIDs() (*gmail.ListMessagesResponse, error) {
	msg_list, err := c.GmailService.Users.Messages.List(c.EmailID).Do()

	return msg_list, err
}
func (c *GmailService) UpdateMsgIDs() ([]*string, error) {
	var updated bool = false
	var updated_msg_list []*string
	msg_list, err := c.GetMsgIDs()
	if err != nil {
		return nil, err
	}
	for _, msg_id := range msg_list.Messages {
		if !c.find_msg(msg_id.Id) {
			if updated != true {
				updated = true
			}
			updated_msg_list = append(updated_msg_list, &msg_id.Id)
		}

	}
	if updated {
		c.ID_DB = *CreateIDList(&msg_list.Messages)

	}
	return updated_msg_list, nil
}

func (c *GmailService) GetMsg(user string, msg_id string) (*gmail.Message, error) {
	msg, err := c.GmailService.Users.Messages.Get(user, msg_id).Do()
	return msg, err

}

func (c *GmailService) GetEmailProfile() (string, error) {
	if c.EmailID == "" {
		usr_name, err := c.GmailService.Users.GetProfile("me").Do()
		if err != nil {
			log.Fatal("Error getting email profile")
		} else {
			c.EmailID = usr_name.EmailAddress
		}
	}

	return c.EmailID, nil
}

func CollectGmailServ(clients []*common.LocalClient, ctx *context.Context, CONFIG_FOLDER *string) ([]*GmailService, error) {
	// var gmail_services []*GmailService
	gmail_services := make([]*GmailService, 0, len(clients))
	for _, client := range clients {
		// client := common.CreateClient(config, tokFile)
		srv, err := client.GetGmailServ(ctx)
		tokFile := client.TK
		db_path := strings.Replace(tokFile, "token_", "id_db_", -1)
		log.Println("Using DB at", db_path)
		for err != nil {
			return nil, err
		}
		var id_db map[string]struct{}
		id_db, err = LoadIDList(db_path)
		if err != nil {
			id_db = make(map[string]struct{})
		}
		client_service := GmailService{
			GmailService: srv,
			ID_DB:        id_db,
			DB_Path:      db_path,
			EmailID:      "",
		}
		client_service.GetEmailProfile()
		log.Println("Successfully created client")
		gmail_services = append(gmail_services, &client_service)
	}
	return gmail_services, nil
}
