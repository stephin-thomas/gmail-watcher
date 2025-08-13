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
	IDDB map[string]struct{}
}

type GmailUserConfig struct {
	EmailID string
	DBPath string
}

func (c *GmailService) Save() error {
	err := io_helpers.SerializeNsave(c.IDDB, c.GmailUserConfig.DBPath)
	if err != nil {
		return fmt.Errorf("failed to save database to %s: %w", c.GmailUserConfig.DBPath, err)
	}
	return nil
}
func (c *GmailService) findMsg(needle string) bool {
	_, found := (c.IDDB)[needle]
	return found
}

func (c *GmailService) GetMsgIDs() (*gmail.ListMessagesResponse, error) {
	log.Printf("Fetching message list for user: %s", c.EmailID)
	msgList, err := c.GmailService.Users.Messages.List(c.EmailID).Do()
	if err != nil {
		log.Printf("ERROR: Failed to list messages for %s: %v", c.EmailID, err)
		return nil, fmt.Errorf("failed to list messages for %s: %w", c.EmailID, err)
	}
	log.Printf("Successfully retrieved %d messages from Gmail API for %s", len(msgList.Messages), c.EmailID)
	return msgList, nil
}
func (c *GmailService) UpdateMsgIDs() ([]*gmail.Message, error) {
	var updated bool
	var updatedMsgList []*gmail.Message
	
	log.Printf("Starting UpdateMsgIDs for user: %s", c.EmailID)
	log.Printf("Current database contains %d message IDs", len(c.IDDB))
	
	msgList, err := c.GetMsgIDs()
	if err != nil {
		log.Printf("ERROR: UpdateMsgIDs failed to get message list: %v", err)
		return nil, err
	}
	
	log.Printf("Comparing %d messages from API with %d in database", len(msgList.Messages), len(c.IDDB))
	
	newMessageCount := 0
	for _, msgID := range msgList.Messages {
		if !c.findMsg(msgID.Id) {
			if !updated {
				updated = true
			}
			updatedMsgList = append(updatedMsgList, msgID)
			newMessageCount++
			log.Printf("Found NEW message ID: %s", msgID.Id)
		}
	}
	
	log.Printf("Found %d new messages out of %d total", newMessageCount, len(msgList.Messages))
	
	if updated {
		log.Printf("Updating database with all %d message IDs", len(msgList.Messages))
		c.IDDB = *CreateIDList(&msgList.Messages)
	} else {
		log.Printf("No new messages found - database unchanged")
	}
	
	return updatedMsgList, nil
}

func (c *GmailService) GetMsg(user string, msgID string) (*gmail.Message, error) {
	msg, err := c.GmailService.Users.Messages.Get(user, msgID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message %s: %w", msgID, err)
	}
	return msg, nil
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

func (clientSrv *GmailService) FetchAllMail(msgIDList []*gmail.Message) (*[]*gmail.Message, error) {
	var allMsgs []*gmail.Message

	log.Printf("Fetching mails for client: %s\n", clientSrv.EmailID)
	for _, msg := range msgIDList {
		msgMail, err := clientSrv.GetMsg("me", msg.Id)
		if err != nil {
			log.Printf("Error getting email %v", err)
			return nil, err
		}
		allMsgs = append(allMsgs, msgMail)
	}
	return &allMsgs, nil
}

func (clientSrv *GmailService) FetchAllMailConcurrent(msgIDList []*gmail.Message) (*[]*gmail.Message, error) {
	log.Println("Fetching mails in parallel")
	var wg sync.WaitGroup
	mailMessage := make(chan *gmail.Message, len(msgIDList))
	// Use a buffered channel to limit concurrency
	semaphore := make(chan struct{}, 4)
	log.Printf("Fetching mails for client: %s\n", clientSrv.EmailID)
	for _, msg := range msgIDList {
		semaphore <- struct{}{}
		wg.Add(1)
		go getMsg(clientSrv, msg, mailMessage, &wg, semaphore)
	}
	// Launch a goroutine to close the channel after sending is done
	go func() {
		wg.Wait() // Wait for all senders to finish
		log.Println("Closing mail message channel")
		close(mailMessage) // Close the channel after all sends are complete
	}()
	allMsgs := readMsgs(mailMessage)

	return &allMsgs, nil
}

func readMsgs(mailMessage chan *gmail.Message) []*gmail.Message {
	var allMsgs []*gmail.Message
	for msgC := range mailMessage {
		allMsgs = append(allMsgs, msgC)
	}
	return allMsgs
}

func getMsg(clientSrv *GmailService, msg *gmail.Message, mailMessage chan *gmail.Message, wg *sync.WaitGroup, semaphore chan struct{}) {
	defer func() {
		<-semaphore
		wg.Done()
	}()

	msgMail, err := clientSrv.GetMsg("me", msg.Id)
	if err != nil {
		log.Printf("Error getting email %s: %v", msg.Id, err)
		return
	}

	select {
	case mailMessage <- msgMail:
		log.Printf("Successfully fetched message %s", msgMail.Id)
	default:
		log.Printf("Channel full, dropping message %s", msgMail.Id)
	}

	// defer func() {
	// 	// Release the token back to the semaphore when the worker is done.

	// }()
}
