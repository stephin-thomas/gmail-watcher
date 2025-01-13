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

type UserCLient struct {
	*http.Client
}

// Retrieve a token, saves the token, then returns the generated client.
func CreateClient(dev_config *oauth2.Config, userTok *oauth2.Token) *UserCLient {
	return &UserCLient{
		dev_config.Client(context.Background(), userTok),
	} // return rq_client, nil
}

func GenNewGmailConfig(dev_config *oauth2.Config, userTok *oauth2.Token, ctx *context.Context) (*gmail_client.GmailUserConfig, error) {
	client := CreateClient(dev_config, userTok)
	srv, err := gmail.NewService(*ctx, option.WithHTTPClient(client.Client))
	if err != nil {
		return nil, fmt.Errorf("error creating new gmail config %w", err)
	}
	email, err := srv.Users.GetProfile("me").Do()
	if err != nil {
		return nil, fmt.Errorf("error getting email address of the profile %w", err)
	}
	db_path := fmt.Sprintf("db_%s.json", uuid.NewString())
	log.Println("Database set as", db_path)
	db_path = path.Join(exports.DATA_FOLDER, db_path)
	return &gmail_client.GmailUserConfig{
		EmailID: email.EmailAddress,
		DB_Path: db_path,
	}, nil
}
func (client *UserCLient) NewGmailService(ctx *context.Context, gmail_user_config gmail_client.GmailUserConfig) (*gmail_client.GmailService, error) {
	srv, err := gmail.NewService(*ctx, option.WithHTTPClient(client.Client))
	if err != nil {
		return nil, fmt.Errorf("unable to create gmail service %w", err)
	}
	var id_db map[string]struct{}
	id_db, err = gmail_client.LoadIDList(gmail_user_config.DB_Path)
	if err != nil {
		id_db = make(map[string]struct{})
		log.Printf("Unable to load id database %v\n Created an empty one in memory instead", err.Error())
	}
	client_service := gmail_client.GmailService{
		GmailService:    srv,
		ID_DB:           id_db,
		GmailUserConfig: gmail_user_config,
	}
	log.Println("Successfully created client")
	return &client_service, nil
}

func (client *UserCLient) NewCalService(ctx *context.Context) (*calendar.Service, error) {
	srv, err := calendar.NewService(*ctx, option.WithHTTPClient(client.Client))
	if err != nil {
		_ = io_helpers.Notify("Unable to retrieve Calendar", "Gmail Watcher Error!")
		return nil, fmt.Errorf("unable to retrieve calendar client: %v", err)
	}
	return srv, nil
}
