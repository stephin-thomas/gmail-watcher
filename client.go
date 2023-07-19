package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/oauth2"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokFile string, web_server *WebServer) *http.Client {
	log.Println("Creating Client")
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		fmt.Println("Error creating client for token file\n", err, "\nGetting new file")
		tok = getTokenFromWeb(config, web_server)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config, web_server *WebServer) *oauth2.Token {
	log.Printf("Getting Token From Web")
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser"+
		"\n%v\n", authURL)
	var authCode string
	if !web_server.server_running {
		web_server.wg.Add(1)
		web_server.web_server = handle_connection(web_server.wg, &web_server.channel_main)
		web_server.server_running = true
		web_server.wg.Wait()
		authCode = <-web_server.channel_main
	}
	web_server.web_server.Close()
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
	srv := &http.Server{Addr: ":5000",
		Handler: http.HandlerFunc(tokenHandler),
	}
	go start_server(srv, wg)
	return srv
}

func start_server(srv *http.Server, wg *sync.WaitGroup) {
	fmt.Println("Starting webserver on port 5000")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("Server returned: %v", err)
	}
	wg.Wait()

}
