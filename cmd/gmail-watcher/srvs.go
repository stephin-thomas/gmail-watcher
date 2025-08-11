package main

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"log"

	"github.com/gmail-watcher/exports"
	"github.com/gmail-watcher/gmail_client"
	"github.com/gmail-watcher/io_helpers"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type UserClient struct {
	*http.Client
}

// Retrieve a token, saves the token, then returns the generated client.
func CreateClient(devConfig *oauth2.Config, userToken *oauth2.Token) *UserClient {
	return &UserClient{
		devConfig.Client(context.Background(), userToken),
	}
}

func GenNewGmailConfig(devConfig *oauth2.Config, userToken *oauth2.Token, ctx *context.Context) (*gmail_client.GmailUserConfig, error) {
	client := CreateClient(devConfig, userToken)
	srv, err := gmail.NewService(*ctx, option.WithHTTPClient(client.Client))
	if err != nil {
		return nil, fmt.Errorf("error creating new gmail config: %w", err)
	}
	email, err := srv.Users.GetProfile("me").Do()
	if err != nil {
		return nil, fmt.Errorf("error getting email address of the profile: %w", err)
	}
	dbPath := fmt.Sprintf("db_%s.json", uuid.NewString())

	log.Println("Database set as", dbPath)

	dbPath = path.Join(exports.DATA_FOLDER, dbPath)
	return &gmail_client.GmailUserConfig{
		EmailID: email.EmailAddress,
		DBPath:  dbPath,
	}, nil
}
func (client *UserClient) NewGmailService(ctx *context.Context, gmailUserConfig gmail_client.GmailUserConfig) (*gmail_client.GmailService, error) {
	srv, err := gmail.NewService(*ctx, option.WithHTTPClient(client.Client))
	if err != nil {
		return nil, fmt.Errorf("unable to create gmail service %w", err)
	}
	var idDB map[string]struct{}
	idDB, err = gmail_client.LoadIDList(gmailUserConfig.DBPath)
	if err != nil {
		idDB = make(map[string]struct{})
		log.Printf("Unable to load id database %v\n Created an empty one in memory instead", err.Error())
	}
	clientService := gmail_client.GmailService{
		GmailService:    srv,
		IDDB:            idDB,
		GmailUserConfig: gmailUserConfig,
	}
	log.Println("Successfully created client")
	return &clientService, nil
}

func (client *UserClient) NewCalService(ctx *context.Context) (*calendar.Service, error) {
	srv, err := calendar.NewService(*ctx, option.WithHTTPClient(client.Client))
	if err != nil {
		_ = io_helpers.Notify("Unable to retrieve Calendar", "Gmail Watcher Error!")
		return nil, fmt.Errorf("unable to retrieve calendar client: %v", err)
	}
	return srv, nil
}
