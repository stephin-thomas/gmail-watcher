package gmail_client

import (
	"fmt"
	"sync"

	"log"

	"github.com/gmail-watcher/io_helpers"
	"google.golang.org/api/gmail/v1"
)

type GmailService struct {
	GmailService *gmail.Service
	GmailUserConfig
	ID_DB map[string]struct{}
}

type GmailUserConfig struct {
	EmailID string
	DB_Path string
}

func (c *GmailService) Save() error {
	err := io_helpers.SerializeNsave(c.ID_DB, c.GmailUserConfig.DB_Path)
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
func (c *GmailService) UpdateMsgIDs() ([]*gmail.Message, error) {
	var updated bool = false
	var updated_msg_list []*gmail.Message
	msg_list, err := c.GetMsgIDs()
	if err != nil {
		return nil, err
	}
	for _, msg_id := range msg_list.Messages {
		if !c.find_msg(msg_id.Id) {
			if !updated {
				updated = true
			}
			updated_msg_list = append(updated_msg_list, msg_id)
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

func (c *GmailService) GetEmailProfile() (*string, error) {
	if c.EmailID == "" {
		usr_name, err := c.GmailService.Users.GetProfile("me").Do()
		if err != nil {
			return nil, fmt.Errorf("error getting email profile %w", err)
		} else {
			c.EmailID = usr_name.EmailAddress
		}
	}

	return &c.EmailID, nil
}

func (client_srv *GmailService) FetchAllMail(msg_id_list []*gmail.Message) (*[]*gmail.Message, error) {
	var all_msgs []*gmail.Message

	log.Printf("Fetching mails for client:- %s\n", client_srv.EmailID)
	for _, msg := range msg_id_list {
		msg_mail, err := client_srv.GetMsg("me", msg.Id)
		if err != nil {
			log.Printf("Error getting email %v", err)
			return nil, err
		}
		all_msgs = append(all_msgs, msg_mail)
	}
	return &all_msgs, nil
}

func (client_srv *GmailService) FetchAllMailConcurrent(msg_id_list []*gmail.Message) (*[]*gmail.Message, error) {
	var wg sync.WaitGroup
	mailMessage := make(chan *gmail.Message)
	// Use a buffered channel to limit concurrency
	semaphore := make(chan struct{}, 4)
	var all_msgs []*gmail.Message
	log.Printf("Fetching mails for client:- %s\n", client_srv.EmailID)
	for _, msg := range msg_id_list {
		semaphore <- struct{}{}
		wg.Add(1)
		go getMsg(client_srv, msg, mailMessage, &wg, &semaphore)
	}
	// Launch a goroutine to close the channel after sending is done
	go func() {
		wg.Wait()                // Wait for all senders to finish
		defer close(mailMessage) // Close the channel after all sends are complete
	}()
	for msg_c := range mailMessage {
		all_msgs = append(all_msgs, msg_c)
	}

	return &all_msgs, nil
}

func getMsg(client_srv *GmailService, msg *gmail.Message, mailMessage chan *gmail.Message, wg *sync.WaitGroup, semaphore *chan struct{}) {
	msg_mail, err := client_srv.GetMsg("me", msg.Id)
	if err != nil {
		log.Printf("Error getting emails:- %v", err)
		fmt.Print("Error getting emails")
	} else {
		mailMessage <- msg_mail
	}
	defer func() {
		// Release the token back to the semaphore when the worker is done.

		wg.Done()
		<-*semaphore
	}()
}
