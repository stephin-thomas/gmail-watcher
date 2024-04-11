package gmail_client

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/gmail-watcher/helpers"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type ClientService struct {
	GmailService *gmail.Service
	EmailID      string
	DB_Path      string
	ID_DB        map[string]struct{}
}
type MailClient interface {
	Save() error
	Update() error
}

func (c *ClientService) Save() error {
	err := helpers.Serialize_n_save(c.ID_DB, c.DB_Path)
	return err
}
func (c *ClientService) find_msg(needle string) bool {
	_, found := (c.ID_DB)[needle]
	return found
}

func (client_srv *ClientService) Update(notify bool) error {
	user := "me"
	updated_msg_list, err := client_srv.UpdateMsgIDs()
	if err != nil {
		return err
	}
	//var updated_emails []string
	log.Printf("Total msgs from google:- %d\n Using only:- 15", len(updated_msg_list))
	// msgs := msg_list.Messages[0:15]
	var max_shown int8 = 15
	var shown_index int8 = 0
	for _, msg_id := range updated_msg_list {
		// if !client_srv.find_msg(*msg_id) {
		shown_index += 1
		msg, err := client_srv.GetMsg(user, *msg_id)
		if err != nil {
			return err
		}
		if max_shown > shown_index && notify {
			helpers.NotifyEmail(&msg.Snippet, &client_srv.EmailID)
		}
		// }
	}
	if shown_index > 0 {
		err := client_srv.Save()
		if err != nil {
			log.Fatalln("Error saving db database", client_srv.DB_Path, err)
		}
	}
	return nil
}

func (c *ClientService) GetMsgIDs() (*gmail.ListMessagesResponse, error) {
	msg_list, err := c.GmailService.Users.Messages.List(c.EmailID).Do()

	return msg_list, err
}
func (c *ClientService) UpdateMsgIDs() ([]*string, error) {
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
		c.ID_DB = make(map[string]struct{})
		for _, msg_id := range msg_list.Messages {
			(c.ID_DB)[msg_id.Id] = struct{}{}

		}

	}
	return updated_msg_list, nil
}

func (c *ClientService) GetMsg(user string, msg_id string) (*gmail.Message, error) {
	msg, err := c.GmailService.Users.Messages.Get(user, msg_id).Do()
	return msg, err

}

func GetEmailProfile(gmail_service *gmail.Service) (*gmail.Profile, error) {
	usr_name, err := gmail_service.Users.GetProfile("me").Do()

	if err != nil {
		log.Fatal("Error getting email profile")

	}
	return usr_name, err
}

func GetGmailServ(client *http.Client, ctx *context.Context) (*gmail.Service, error) {
	srv, err := gmail.NewService(*ctx, option.WithHTTPClient(client))
	return srv, err

}
func CollectGmailServ(config *oauth2.Config, ctx *context.Context, tokFiles *[]string, CONFIG_FOLDER *string) ([]*ClientService, error) {
	log.Println("Collecting Gmail Clients from configuration from tokens", tokFiles)
	var gmail_services []*ClientService

	for _, tokFile := range *tokFiles {
		client := GetClient(config, tokFile)
		srv, err := GetGmailServ(client, ctx)
		email, err := GetEmailProfile(srv)

		db_path := strings.Replace(tokFile, "token_", "id_db_", -1)
		// db = fmt.Sprintf("id_db_%s")
		// db_file := fmt.Sprintf("id_db_%s.json", email.EmailAddress)
		// db := path.Join(*CONFIG_FOLDER, db_file)
		log.Println("Using DB at", db_path)
		for err != nil {
			return nil, err
		}
		id_db, err := helpers.Load_old_msg_ids(db_path)
		if err != nil {
			id_db = make(map[string]struct{})
		}
		client_service := ClientService{
			GmailService: srv,
			ID_DB:        id_db,
			DB_Path:      db_path,
			EmailID:      email.EmailAddress,
		}
		log.Println("Successfully created client")
		gmail_services = append(gmail_services, &client_service)
		//get_email(&client_service)
	}
	return gmail_services, nil
}
