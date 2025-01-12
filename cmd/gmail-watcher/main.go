package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"

	"google.golang.org/api/calendar/v3"

	"github.com/alecthomas/kong"
	"github.com/gmail-watcher/auth"
	"github.com/gmail-watcher/daemon"
	"github.com/gmail-watcher/exports"
	"github.com/gmail-watcher/gcalendar"
	"github.com/gmail-watcher/gmail_client"
	"github.com/gmail-watcher/io_helpers"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

func init() {
	// var logFile *os.File
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	err := io_helpers.CreateFolder(exports.DATA_FOLDER)
	if err != nil {
		log.Fatalf("error creating folder %v %v", exports.DATA_FOLDER, err)
	}
	logFile, err := os.OpenFile(exports.LOG_FILE_PATH, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("error opening log file %v", err)
	}
	// Redirect log output to the file
	log.SetOutput(logFile)
	err = io_helpers.CreateFolder(exports.CONFIG_FOLDER)
	if err != nil {
		log.Fatalf("error creating folder %v %v", exports.CONFIG_FOLDER, err)
	}
	err = io_helpers.CreateFolder(exports.ASSETS_PATH)
	if err != nil {
		log.Fatalf("error creating folder %v %v", exports.ASSETS_PATH, err)
	}
	err = io_helpers.CopyAssets(exports.ASSETS_SOURCE_PATH, exports.ASSETS_PATH)
	if err != nil {
		log.Println("Error copying assets", err)
	}
	fmt.Println("Log file set as ", exports.LOG_FILE_PATH)
	//This is a temporary function to copy assets. Should be removed when assets folders are created by the installation
	log.Println("Config Folder:-", exports.CONFIG_FOLDER)
	log.Println("Data assets Folder:-", exports.ASSETS_PATH)
	err = io_helpers.CopyAssets(exports.ASSETS_SOURCE_PATH, exports.ASSETS_PATH)
	if err != nil {
		log.Println("Error copying assets ", err)
	}
}

func main() {
	runtime.GOMAXPROCS(1) // Restrict to a single OS thread
	log.Printf("Goroutines: %d\n", runtime.NumGoroutine())
	// fmt.Printf("Threads: %d\n", runtime.NumThread())
	ctx := context.Background()

	// Parse command line arguments into the Config struct.
	cli_ctx := kong.Parse(&CLI,
		kong.Name("gmail-watcher"),
		kong.Description("A simple CLI application to show new email notifications from gmail"),
		kong.UsageOnError(), // Show usage on error

	)

	// Handle errors during parsing.
	if cli_ctx.Error != nil {
		fmt.Println("Error parsing arguments:", cli_ctx.Error)
		os.Exit(1)
	}

	cli_cmd := cli_ctx.Command()

	dev_creds_json, err := os.ReadFile(exports.CREDENTIALS_FILE)
	showApiEnableInstructions(err)
	// If modifying these scopes, delete your previously saved token.json.
	Devconfig, err := google.ConfigFromJSON(dev_creds_json, gmail.GmailReadonlyScope, calendar.CalendarReadonlyScope)
	showApiEnableInstructions(err)
	if cli_cmd == "login auth" {
		var app_config AppConfig
		// var user_configs *[]UserConfig
		app_config_ptr, err := LoadAppConfig()

		if err != nil {
			log.Println("Error reading app config creating new in memory")
			user_configs := make([]UserConfig, 0, 1)
			app_config = AppConfig{UserConfigs: &user_configs}
		} else {
			app_config = *app_config_ptr
		}
		token, err := auth.GetUserTokenFromWeb(Devconfig, CLI.Login.Auth.AuthPort)
		if err != nil {
			_ = io_helpers.Notify("Error getting token from web", "Gmail Watcher: Error!")
			log.Fatalf("error getting token from web %v", err)
		}
		gmailUserConfig, err := GenNewGmailConfig(Devconfig, token, &ctx)
		if err != nil {
			log.Fatalf("Unable to generate gmail user config %v", err)
		}
		newUserConfig := UserConfig{
			UserToken:       token,
			GmailUserConfig: *gmailUserConfig,
		}
		*app_config.UserConfigs = append(*app_config.UserConfigs, newUserConfig)
		fmt.Printf("App Config %v\n", app_config)
		err = app_config.Save()
		if err != nil {
			log.Fatalln(err.Error())
		}
		return

	}
	app_config_ptr, err := LoadAppConfig()
	if err != nil {
		fmt.Println("No logins found. Login and try again")
		log.Fatalf("Couldn't read app config %v", err)
	}
	app_config := *app_config_ptr
	gmailServices := make([]*gmail_client.GmailService, 0, len(*app_config.UserConfigs))
	calServices := make([]*calendar.Service, 0, len(*app_config.UserConfigs))
	// var gmailServices *[]*gmail_client.GmailService
	// var calServices *[]*calendar.Service
	if isGmailSubCommand(cli_cmd) {
		for _, userConfig := range *app_config.UserConfigs {
			gmailUserClient := CreateClient(Devconfig, userConfig.UserToken)
			gmailService, err := gmailUserClient.NewGmailService(&ctx, userConfig.GmailUserConfig)
			if err != nil {
				log.Fatalf("Error generating gmail service %v\n", err)
			}
			gmailServices = append(gmailServices, gmailService)
		}
	} else if isCalSubCommand(cli_cmd) {
		for _, userConfig := range *app_config.UserConfigs {
			calUserClient := CreateClient(Devconfig, userConfig.UserToken)
			calService, err := calUserClient.NewCalService(&ctx)
			if err != nil {
				log.Fatalf("Error generating gmail service %v\n", err)
			}
			calServices = append(calServices, calService)
		}
	}

	log.Println("cli_cmd", cli_cmd)

	switch cli_cmd {
	case "login delete":
		{
			if CLI.Login.Delete.All && len(CLI.Login.Delete.Index) > 0 {
				fmt.Println("Error: Only one of --all or --index can be specified")
				os.Exit(1)
			}
			if CLI.Login.Delete.All {
				var userconfigs []UserConfig
				userconfigs = make([]UserConfig, 0, 1)
				app_config.UserConfigs = &userconfigs
			} else {
				for _, login_index := range CLI.Login.Delete.Index {
					if login_index == 0 || login_index > len(CLI.Login.Delete.Index) {
						fmt.Println("No user at index", login_index)
					}
					fmt.Println("Removing", login_index, (*app_config.UserConfigs)[login_index-1].EmailID)
					deleteSliceIndex(app_config.UserConfigs, login_index-1)
				}
			}
			err = app_config.Save()
			if err != nil {
				log.Fatalln(err.Error())
			}
		}
	case "login list-users":
		{
			for index, user := range *app_config.UserConfigs {
				fmt.Println(index+1, user.EmailID)

			}
		}

	case "gmail list":
		{
			// parallel := CLI.Gmail.List.Parallel
			// list_len := 15
			var msgs *[]*gmail.Message

			for _, client_srv := range gmailServices {
				list_len := CLI.Gmail.List.ListLen
				msg_id_list, err := client_srv.GetMsgIDs()

				if err != nil {
					fmt.Println("Error getting emails msg ids")
					log.Fatalf("error getting emails msg ids:- %v", err)
				}

				if CLI.Gmail.List.Parallel {
					msgs, err = client_srv.FetchAllMailConcurrent(msg_id_list.Messages[:list_len])
					if err != nil {
						log.Fatalf("error \n%s", err)
					}

				} else {
					msgs, err = client_srv.FetchAllMail(msg_id_list.Messages[:list_len])
					if err != nil {
						log.Fatalf("error \n%s", err)
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
			err = daemon.RunDaemon(CLI.Gmail.Daemon.Retries, CLI.Gmail.Daemon.MaxNotifications, &gmailServices)
			if err != nil {
				_ = io_helpers.Notify("Error occured", "Gmail Watcher: Error!")
				log.Fatalf("gmail daemon failed %v\n", err)
			}
		}
	case "cal":
		{
			// calendar_services := make([]string, len(clients))
			for _, cal_srv := range calServices {
				if err != nil {
					return
				}
				err = gcalendar.GetEvents(cal_srv, CLI.Cal.MaxResults)
				if err != nil {
					log.Fatalf("error occured when getting events %v", err)
				}
			}
		}
	default:
		fmt.Println("unknown Command", cli_cmd)
		log.Printf("unknown Command %v", cli_cmd)
		panic(cli_ctx.Command())
	}
}

func showApiEnableInstructions(err error) {
	if err != nil {
		fmt.Printf("Unable parse secret file:\n Follow the steps 'Enable the API' and 'Authorize credentials for a desktop application' from the following page\n https://developers.google.com/gmail/api/quickstart/go \n Note:- Ignore all other steps\n rename the downloaded file to credentials.json and copy it to ~/.config/gmail_watcher\n")
		fmt.Printf("Make sure gmail readonlyscope and calendar readonly scope is available for the account\n%s", err)
		log.Fatalf("%v:\nUnable to parse client secret file to config or obtain permissions\n", err)
	}
}
