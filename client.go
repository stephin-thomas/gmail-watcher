package main

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"sync"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokFile string, web_server *WebServer) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	// tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config, web_server)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config, web_server *WebServer) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"\n%v\n", authURL)
	var authCode string
	// var c chan string
	if web_server.server_running == false {
		// var wg sync.WaitGroup
		web_server.wg.Add(1)
		web_server.web_server = handle_connection(web_server.wg, &web_server.channel_main)
		// fmt.Println("Webserver Created and returned to the main function")
		web_server.server_running = true
		web_server.wg.Wait()
		authCode = <-web_server.channel_main
		// fmt.Println("Found Auth Code")
	} else {
		// var nwg sync.WaitGroup
		web_server.wg.Add(1)
		web_server.wg.Wait()
		authCode = <-web_server.channel_main
		web_server.web_server.Close()
	}
  // authCodeM :=authCode[2:]
	tok, err := config.Exchange(context.TODO(), authCode)
	// tok, err := config.Exchange(oauth2.NoContext, authCode)
	
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func handle_connection(wg *sync.WaitGroup, c *chan string) *http.Server {
	tokenHandler := func(w http.ResponseWriter, req *http.Request) {
		authCode := req.URL.Query().Get("code")
		*c <- authCode
		io.WriteString(w, "Your Gmail Authenticated you could close the browser now!\n")
		defer wg.Done() // let main know we are done cleaning up
	}
	srv := &http.Server{Addr: ":5000",
		Handler: http.HandlerFunc(tokenHandler),
	}

	go func() {
		
		fmt.Println("Starting webserver on port 5000")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Server returned: %v", err)
		}
	}()

	return srv
}

