package common

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gmail-watcher/io_helpers"
	"github.com/gmail-watcher/paths"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type LocalClient struct {
	*http.Client
}

func (client *LocalClient) GetGmailServ(ctx *context.Context) (*gmail.Service, error) {
	srv, err := gmail.NewService(*ctx, option.WithHTTPClient(client.Client))
	return srv, err

}

// Retrieve a token, saves the token, then returns the generated client.
func CreateClient(config *oauth2.Config, tokFile string) LocalClient {
	log.Println("Creating Client")
	tok, err := io_helpers.TokenFromFile(tokFile)
	if err != nil {
		log.Println("Error creating client for token file\n", err, "\nGetting new file")
		tok = GetTokenFromWeb(config)
		io_helpers.SaveToken(tokFile, tok)
	}
	return LocalClient{config.Client(context.Background(), tok)}
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
