package auth

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/gmail-watcher/io_helpers"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

func ChangeServerPort(config *oauth2.Config, port uint64) error {
	fmt.Println("server listening on ", port)
	finalURl, err := url.Parse(config.RedirectURL)
	if err != nil {
		return fmt.Errorf("error parsing redirect url from config \n%s \n%w", config.RedirectURL, err)
	}
	scheme := finalURl.Scheme
	config_host := finalURl.Hostname()
	config_port := finalURl.Port()
	req_port := fmt.Sprintf("%d", port)
	if config_port != req_port {
		config.RedirectURL = fmt.Sprintf("%s://%s:%d", scheme, config_host, port)
		log.Printf("Redirect url set as %s", config.RedirectURL)
		// if err != nil {
		// 	return fmt.Errorf("error replacing port in the config %s", err)
		// }
		// err := io_helpers.SerializeNsave(*creds, paths.CREDENTIALS_FILE)
		// if err != nil {
		// 	return fmt.Errorf("Error serializing and saving the newly generated json %s", err)
		// }
	}
	return nil
}

// Request a token from the web, then returns the retrieved token.
func GetUserTokenFromWeb(app_config *oauth2.Config, callback_port uint64) (*oauth2.Token, error) {
	err := ChangeServerPort(app_config, callback_port)
	if err != nil {
		return nil, fmt.Errorf("error changing server port %w", err)
	}
	log.Println("Getting token from web")
	channel := make(chan string, 1)
	wg := sync.WaitGroup{}
	log.Printf("Getting Token From Web")
	authURL := app_config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	_ = browser.OpenURL(authURL)
	fmt.Printf("Go to the following link in your browser"+
		"\n%v\n", authURL)
	var authCode string
	wg.Add(1)
	server_url, _ := url.Parse(app_config.RedirectURL)
	server_host := server_url.Hostname()
	server_port := server_url.Port()
	server_url_final := fmt.Sprintf("%s:%s", server_host, server_port)
	web_server := handle_connection(&wg, &channel, server_url_final)
	wg.Wait()
	authCode = <-channel
	defer func() {
		err := web_server.Close()
		log.Printf("Error occured when closing the web browser %v", err)
	}()
	tok, err := app_config.Exchange(context.TODO(), authCode)
	if err != nil {
		_ = io_helpers.Notify("Unable to retrieve web token", "error!")
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

func handle_connection(wg *sync.WaitGroup, c *chan string, redirect_url string) *http.Server {
	log.Println("Getting code from response")
	tokenHandler := func(w http.ResponseWriter, req *http.Request) {
		authCode := req.URL.Query().Get("code")
		*c <- authCode
		_, err := io.WriteString(w, "Your Gmail Authenticated you could close the browser now!\n")
		if err != nil {
			log.Printf("error when writing the confirmation msg to browser %v\n", err)
		}
		defer wg.Done() // let main know we are done
	}
	srv := &http.Server{Addr: redirect_url,
		Handler: http.HandlerFunc(tokenHandler),
	}
	go start_server(srv)
	return srv
}

func start_server(srv *http.Server) error {
	log.Printf("Starting webserver on:-%s\n", srv.Addr)
	err := srv.ListenAndServe()
	switch err {
	case http.ErrServerClosed:
		log.Printf("Server closed successfully: %v", err)
		return nil
	default:
		log.Fatalf("error: unable to serve %v", err)
		return fmt.Errorf("error:- unable to serve: %w", err)
	}

}
