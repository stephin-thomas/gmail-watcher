package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"

	"google.golang.org/api/calendar/v3"

	"github.com/alecthomas/kong"
	"github.com/gmail-watcher/common"
	"github.com/gmail-watcher/daemon"
	"github.com/gmail-watcher/gcalendar"
	"github.com/gmail-watcher/gmail_client"
	"github.com/gmail-watcher/io_helpers"
	"github.com/gmail-watcher/paths"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

var logFile *os.File

func init() {
	if !io_helpers.CreateFolder(paths.CONFIG_FOLDER) {
		log.Println("Config folder found")
	}
	logFile, err := os.OpenFile(paths.LOG_FILE_PATH, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	fmt.Printf("Log file set as %s\n", paths.LOG_FILE_PATH)
	//This is a temporary function to copy assets. Should be removed when assets folders are created by the installation
	if err != nil {
		log.Println("Error opening log file %w", err)
		log.Fatalf("Error opening log file ")
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Redirect log output to the file
	log.SetOutput(logFile)
	log.Println("Config Folder:-", paths.CONFIG_FOLDER)
	io_helpers.CopyAssets(paths.ASSETS_SOURCE_PATH, paths.ASSETS_PATH)
}

func main() {
	runtime.GOMAXPROCS(1) // Restrict to a single OS thread
	log.Printf("Goroutines: %d\n", runtime.NumGoroutine())
	// fmt.Printf("Threads: %d\n", runtime.NumThread())

	defer logFile.Close()
	log.Println("setup complete")
	ctx := context.Background()
	config_json, err := os.ReadFile(paths.CREDENTIALS_FILE)
	if err != nil {
		log.Fatalf("Error reading credentials.json file \n%s", err)
	}
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(config_json, gmail.GmailReadonlyScope, calendar.CalendarReadonlyScope)
	if err != nil {
		fmt.Printf(`
		Unable parse secret file:
		 - Follow the steps 'Enable the API' and 'Authorize credentials for a desktop application' from the following page
		  https://developers.google.com/gmail/api/quickstart/go 
			  ( Note:- 
			  	Ignore all other steps 
				Make sure gmail readonlyscope and calendar readonly scope is available for the account
			  )
		  - Rename the downloaded file to credentials.json and copy it to ~/.config/gmail_watcher

		  Check the logs for more info
		`)
		log.Fatalf("%v:\nUnable to parse client secret file to config or obtain permissions\n", err)
	}

	var clients []*common.LocalClient
	//if tokfiles exist load that and add the new token on login. Or use the pre-existing token files to access the apis
	tokFiles, err := io_helpers.LoadJsonList(paths.LOGIN_TOKENS_LIST_FILE)
	log.Printf("%d token files found\n", len(tokFiles))
	if err != nil {
		log.Println("Error getting token files. %w", err)
		tokFiles = make([]string, 0)
	}
	clients = make([]*common.LocalClient, 0, len(tokFiles))
	cli_ctx := kong.Parse(&CLI)
	if cli_ctx.Command() != "login" {
		// var clients []*common.LocalClient
		for _, tk := range tokFiles {
			client := common.CreateClient(config, tk)
			clients = append(clients, &client)
		}
		if len(clients) == 0 {
			fmt.Println("Error generating clients, No clients found")
			log.Fatalln("Error generating clients, No clients found")
		}

	}
	// var max_retries uint8 = 3
	max_retries := CLI.Gmail.MaxRetries
	cli_cmd := cli_ctx.Command()
	clientSrvs := confirm_logins(ctx, &cli_cmd, clients, max_retries)
	switch cli_cmd {
	case "login":
		{
			err = gmail_client.ChangeServerPort(config, CLI.Login.AuthPort)
			if err != nil {
				log.Printf("Error changing server port\n%s", err)
			}
			token := common.GetTokenFromWeb(config)
			token_file_path, err := gmail_client.AddToken(&tokFiles)
			if err != nil {
				fmt.Printf("Error Occured")
				log.Printf("Error adding token :- %v", err)
			}
			io_helpers.SaveToken(*token_file_path, token)
		}

	case "gmail list":
		{
			// parallel := CLI.Gmail.List.Parallel
			// list_len := 15
			var msgs *[]*gmail.Message
			for _, client_srv := range clientSrvs {
				list_len := CLI.Gmail.List.ListLen
				msg_id_list, err := client_srv.GetMsgIDs()

				if err != nil {
					fmt.Println("Error getting emails msg ids")
					log.Fatalf("Error getting emails msg ids:- %v", err)
				}

				if CLI.Gmail.List.Parallel {
					msgs, err = gmail_client.FetchMailConcurrent(client_srv, msg_id_list.Messages[:list_len])
					if err != nil {
						log.Fatalf("Error \n%s", err)
					}

				} else {
					msgs, err = gmail_client.FetchMail(client_srv, msg_id_list.Messages[:list_len])
					if err != nil {
						log.Fatalf("Error \n%s", err)
					}

				}

				for _, msg := range *msgs {
					fmt.Println("-----")
					sender := gmail_client.GetSender(msg)
					fmt.Printf("%s\n", sender)
					fmt.Printf("%s\n", msg.Snippet)
					fmt.Println("-----")
					// panic("panicked")
				}
			}
			return
		}
	case "gmail daemon":
		{
			daemon.RunDaemon(CLI.Gmail.MaxRetries, CLI.Gmail.Daemon.MaxNotifications, clientSrvs)
		}
	default:
		fmt.Println("Unknown Command")
		panic(cli_ctx.Command())
	case "cal":
		{
			// calendar_services := make([]string, len(clients))
			for _, client := range clients {
				cal_srv, err := client.GetCalSrv(&ctx)
				if err != nil {
					return
				}
				gcalendar.GetEvents(cal_srv, CLI.Cal.MaxResults)
			}
		}
	}
}

func confirm_logins(ctx context.Context, cli_cmd *string, clients []*common.LocalClient, max_retries uint8) []*gmail_client.GmailService {
	var clientSrvs []*gmail_client.GmailService
	var loginNotFound bool
	if *cli_cmd != "login" {
		clientSrvs, loginNotFound = gmail_client.GetClientSrvs(clients, ctx, max_retries)
		if loginNotFound {
			log.Fatalf("Login not found when running daemon")
		}

	}
	return clientSrvs
}
