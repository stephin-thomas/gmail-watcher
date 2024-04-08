package gmail_client

import (
	"fmt"
	"log"

	"github.com/gen2brain/beeep"
	"github.com/gmail-watcher/helpers"
	"github.com/gmail-watcher/paths"
	"google.golang.org/api/gmail/v1"
)

type ClientService struct {
	GmailService *gmail.Service
	EmailID      string
	DB           string
	ID_DB        *map[string]struct{}
}

func (c ClientService) save() error {
	err := helpers.Serialize_n_save(c.ID_DB, c.DB)
	return err
}
func (c ClientService) find_msg(needle string) bool {
	_, found := (*c.ID_DB)[needle]
	return found
}

func (client_srv *ClientService) Run(notify bool) ([]*gmail.Message, error) {
	user := "me"
	msg_list, err := client_srv.GetMsgIDs(user)
	if err != nil {
		return nil, err
	}
	//var updated_emails []string
	log.Printf("Total msgs from google:- %d\n Using only:- 15", len(msg_list.Messages))
	// msgs := msg_list.Messages[0:15]
	msgs := msg_list.Messages
	var max_shown int8 = 15
	var shown_index int8 = 0
	for _, msg := range msgs {
		if client_srv.find_msg(msg.Id) {
			shown_index = shown_index + 1
			msg, err := client_srv.GetMsg(user, msg.Id)
			if err != nil {
				return nil, err
			}
			if err == nil {
				if max_shown > shown_index && notify {
					show_emails(msg, &client_srv.EmailID)
				}
			} else {
				log.Fatalf("error occured getting email %v", err)
			}
		}
	}
	if shown_index > 0 {
		err := client_srv.save()
		if err != nil {
			log.Fatalln("Error saving db database", client_srv.DB, err)
		}
	}
	return msgs, nil
}

func show_emails(msg *gmail.Message, user_email *string) {
	err := beeep.Notify(fmt.Sprintf("Gmail Watcher:-%s", *user_email), msg.Snippet, paths.NOTIFICATION_ICON)
	if err != nil {
		log.Println("Error during notification", err)
	}
}

func (c *ClientService) GetMsgIDs(user string) (*gmail.ListMessagesResponse, error) {
	msg_list, err := c.GmailService.Users.Messages.List(user).Do()
	return msg_list, err
}

func (c *ClientService) GetMsg(user string, msg_id string) (*gmail.Message, error) {
	msg, err := c.GmailService.Users.Messages.Get(user, msg_id).Do()
	return msg, err

}
