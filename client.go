package main

import (
	"context"
	"fmt"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"sync"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokFile string) *http.Client {
	log.Println("Creating Client")
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		log.Println("Error creating client for token file\n", err, "\nGetting new file")
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
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
	srv := &http.Server{Addr: fmt.Sprintf(":%d", PORT),
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
