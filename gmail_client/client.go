package gmail_client

import (
	"strings"

	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gmail-watcher/helpers"
	"github.com/gmail-watcher/paths"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func GetClient(config *oauth2.Config, tokFile string) *http.Client {
	log.Println("Creating Client")
	tok, err := helpers.TokenFromFile(tokFile)
	if err != nil {
		log.Println("Error creating client for token file\n", err, "\nGetting new file")
		tok = GetTokenFromWeb(config)
		helpers.SaveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func GetTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	channel := make(chan string, 1)
	wg := &sync.WaitGroup{}
	log.Printf("Getting Token From Web")
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	browser.OpenURL(authURL)
	fmt.Printf("Go to the following link in your browser"+
		"\n%v\n", authURL)
	var authCode string
	wg.Add(1)
	web_server := handle_connection(wg, &channel)
	wg.Wait()
	authCode = <-channel
	web_server.Close()
	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func handle_connection(wg *sync.WaitGroup, c *chan string) *http.Server {
	log.Println("Getting code from response")
	tokenHandler := func(w http.ResponseWriter, req *http.Request) {
		authCode := req.URL.Query().Get("code")
		*c <- authCode
		io.WriteString(w, "Your Gmail Authenticated you could close the browser now!\n")
		defer wg.Done() // let main know we are done
	}
	srv := &http.Server{Addr: fmt.Sprintf(":%d", paths.PORT),
		Handler: http.HandlerFunc(tokenHandler),
	}
	go start_server(srv)
	return srv
}

func start_server(srv *http.Server) {
	log.Println("Starting webserver on port 5000")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("Server returned: %v", err)
	}

}

type ClientService struct {
	GmailService *gmail.Service
	EmailID      string
	DB_Path      string
	ID_DB        map[string]struct{}
}

func (c *ClientService) Save() error {
	err := helpers.Serialize_n_save(c.ID_DB, c.DB_Path)
	return err
}
func (c *ClientService) find_msg(needle string) bool {
	_, found := (c.ID_DB)[needle]
	return found
}

// func (client_srv *ClientService) Update(notify bool) error {
// 	user := "me"
// 	updated_msg_list, err := client_srv.UpdateMsgIDs()
// 	if err != nil {
// 		return err
// 	}
// 	//var updated_emails []string
// 	log.Printf("Total msgs from google:- %d\n Using only:- 15", len(updated_msg_list))
// 	// msgs := msg_list.Messages[0:15]
// 	var max_shown int8 = 15
// 	var shown_index int8 = 0
// 	for _, msg_id := range updated_msg_list {
// 		// if !client_srv.find_msg(*msg_id) {
// 		shown_index += 1
// 		msg, err := client_srv.GetMsg(user, *msg_id)
// 		if err != nil {
// 			return err
// 		}
// 		if max_shown > shown_index && notify {
// 			helpers.Notify(&msg.Snippet, &client_srv.EmailID)
// 		}
// 		// }
// 	}
// 	if shown_index > 0 {
// 		err := client_srv.Save()
// 		if err != nil {
// 			log.Fatalln("Error saving db database", client_srv.DB_Path, err)
// 		}
// 	}
// 	return nil
// }

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
		c.ID_DB = *CreateIDList(&msg_list.Messages)
		// c.ID_DB = make(map[string]struct{})
		// for _, msg_id := range msg_list.Messages {
		// 	(c.ID_DB)[msg_id.Id] = struct{}{}

		// }

	}
	return updated_msg_list, nil
}

func (c *ClientService) GetMsg(user string, msg_id string) (*gmail.Message, error) {
	msg, err := c.GmailService.Users.Messages.Get(user, msg_id).Do()
	return msg, err

}

func (c *ClientService) GetEmailProfile() (string, error) {
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

		db_path := strings.Replace(tokFile, "token_", "id_db_", -1)
		// db = fmt.Sprintf("id_db_%s")
		// db_file := fmt.Sprintf("id_db_%s.json", email.EmailAddress)
		// db := path.Join(*CONFIG_FOLDER, db_file)
		log.Println("Using DB at", db_path)
		for err != nil {
			return nil, err
		}
		var id_db map[string]struct{}
		// id_db map[string]struct{} , err error := gmail_client.Load_old_msg_ids(db_path)
		id_db, err = LoadIDList(db_path)
		if err != nil {
			id_db = make(map[string]struct{})
		}
		client_service := ClientService{
			GmailService: srv,
			ID_DB:        id_db,
			DB_Path:      db_path,
			EmailID:      "",
		}
		client_service.GetEmailProfile()
		log.Println("Successfully created client")
		gmail_services = append(gmail_services, &client_service)
		//get_email(&client_service)
	}
	return gmail_services, nil
}
