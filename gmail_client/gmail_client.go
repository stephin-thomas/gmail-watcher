package gmail_client

import (
	"fmt"
	"log"

	"github.com/gen2brain/beeep"
	"google.golang.org/api/gmail/v1"
)

type clientService struct {
	gmail_service *gmail.Service
	email_id      string
	db            string
	id_db         *map[string]struct{}
}

func (c clientService) find_msg(needle string) bool {
	_, found := (*c.id_db)[needle]
	return found
}

func (client_srv *clientService) run(notify bool) ([]*gmail.Message, error) {
	user := "me"
	msg_list, err := get_msg_ids(client_srv.gmail_service, user)
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
			msg, err := get_msg(client_srv.gmail_service, user, msg.Id)
			if err != nil {
				return nil, err
			}
			if err == nil {
				if max_shown > shown_index && notify {
					show_emails(msg, &client_srv.email_id)
				}
			} else {
				log.Fatalf("error occured getting email %v", err)
			}
		}
	}
	if shown_index > 0 {
		err := client_srv.save()
		if err != nil {
			log.Fatalln("Error saving db database", client_srv.db, err)
		}
	}
	return msgs, nil
}

func show_emails(msg *gmail.Message, user_email *string) {
	err := beeep.Notify(fmt.Sprintf("Gmail Watcher:-%s", *user_email), msg.Snippet, NOTIFICATION_ICON)
	if err != nil {
		log.Println("Error during notification", err)
	}
}

func get_msg_ids(srv *gmail.Service, user string) (*gmail.ListMessagesResponse, error) {
	msg_list, err := srv.Users.Messages.List(user).Do()
	return msg_list, err
}

func get_msg(srv *gmail.Service, user string, msg_id string) (*gmail.Message, error) {
	msg, err := srv.Users.Messages.Get(user, msg_id).Do()
	return msg, err

}
