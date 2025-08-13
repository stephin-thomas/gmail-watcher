package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

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

	// log.Println("DB file set as:-", exports.CREDENTIALS_FILE)
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
	if len(*app_config.UserConfigs) == 0 {
		fmt.Println("No users found")
		return
	}

	gmailServices := make([]*gmail_client.GmailService, 0, len(*app_config.UserConfigs))
	calServices := make([]*calendar.Service, 0, len(*app_config.UserConfigs))
	// var gmailServices *[]*gmail_client.GmailService
	// var calServices *[]*calendar.Service
	if isGmailSubCommand(cli_cmd) || cli_cmd == "daemon" {
		log.Printf("Creating Gmail services for command: %s", cli_cmd)
		configUpdated := false
		for i, userConfig := range *app_config.UserConfigs {
			gmailUserClient := CreateClient(Devconfig, userConfig.UserToken)
			gmailService, err := gmailUserClient.NewGmailService(&ctx, userConfig.GmailUserConfig)
			if err != nil {
				log.Fatalf("Error generating gmail service %v\n", err)
			}
			
			// Update config if database path was generated
			if (*app_config.UserConfigs)[i].GmailUserConfig.DBPath != gmailService.GmailUserConfig.DBPath {
				(*app_config.UserConfigs)[i].GmailUserConfig.DBPath = gmailService.GmailUserConfig.DBPath
				configUpdated = true
				log.Printf("Updated config with DB path for %s", userConfig.EmailID)
			}
			
			gmailServices = append(gmailServices, gmailService)
			log.Printf("Created Gmail service for: %s", userConfig.EmailID)
		}
		
		// Save config if any database paths were updated
		if configUpdated {
			err = app_config.Save()
			if err != nil {
				log.Printf("Warning: Failed to save updated config: %v", err)
			} else {
				log.Println("Saved updated config with database paths")
			}
		}
		
		log.Printf("Total Gmail services created: %d", len(gmailServices))
	}
	
	if isCalSubCommand(cli_cmd) || cli_cmd == "daemon" {
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
	case "daemon":
		{
			// Run unified daemon with configurable services
			log.Printf("Starting unified notification daemon")
			log.Printf("Gmail enabled: %t, Calendar enabled: %t", CLI.Daemon.GmailEnabled, CLI.Daemon.CalendarEnabled)
			
			if !CLI.Daemon.GmailEnabled && !CLI.Daemon.CalendarEnabled {
				log.Fatalf("At least one service (Gmail or Calendar) must be enabled")
			}
			
			_ = io_helpers.Notify("Unified Daemon Started", "Gmail Watcher")
			
			// Start Gmail daemon if enabled
			if CLI.Daemon.GmailEnabled {
				go func() {
					log.Printf("Starting Gmail monitoring with %d retries, %d max notifications", CLI.Daemon.GmailRetries, CLI.Daemon.GmailMaxNotifications)
					err := daemon.RunDaemon(CLI.Daemon.GmailRetries, CLI.Daemon.GmailMaxNotifications, &gmailServices)
					if err != nil {
						log.Printf("Gmail daemon error: %v", err)
						_ = io_helpers.Notify("Gmail daemon error", "Gmail Watcher: Error!")
					}
				}()
			}
			
			// Start Calendar daemon if enabled
			if CLI.Daemon.CalendarEnabled {
				for _, calSrv := range calServices {
					calService := gcalendar.NewCalendarService(calSrv)
					go func(cs *gcalendar.CalendarService) {
						log.Printf("Starting Calendar monitoring with %d minute intervals, notifications at %v minutes before", CLI.Daemon.CalendarCheckInterval, CLI.Daemon.CalendarNotifyBefore)
						err := cs.RunCalendarDaemon(CLI.Daemon.CalendarCheckInterval, CLI.Daemon.CalendarNotifyBefore)
						if err != nil {
							log.Printf("Calendar daemon error: %v", err)
						}
					}(calService)
				}
			}
			
			log.Printf("Unified daemon running - monitoring enabled services")
			
			// Set up graceful shutdown
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			
			// Wait for shutdown signal
			<-sigCh
			log.Println("Shutdown signal received, stopping daemon gracefully...")
			_ = io_helpers.Notify("Daemon Shutting Down", "Gmail Watcher")
		}
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

			for _, clientSrv := range gmailServices {
				listLen := CLI.Gmail.List.ListLen
				msgIDList, err := clientSrv.GetMsgIDs()

				if err != nil {
					fmt.Println("Error getting emails msg ids")
					log.Fatalf("error getting emails msg ids:- %v", err)
				}

				if CLI.Gmail.List.Parallel {
					msgs, err = clientSrv.FetchAllMailConcurrent(msgIDList.Messages[:listLen])
					if err != nil {
						log.Fatalf("error \n%s", err)
					}

				} else {
					msgs, err = clientSrv.FetchAllMail(msgIDList.Messages[:listLen])
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
			// Show upcoming calendar events
			for _, calSrv := range calServices {
				calService := gcalendar.NewCalendarService(calSrv)
				
				// Get events for the specified number of days
				endTime := time.Now().AddDate(0, 0, CLI.Cal.Days)
				events, err := calService.GetEventsInRange(time.Now(), endTime, CLI.Cal.MaxResults)
				if err != nil {
					log.Fatalf("error getting calendar events: %v", err)
				}
				
				if len(events) == 0 {
					fmt.Println("No upcoming events found.")
					continue
				}
				
				// Display events in an organized format
				fmt.Printf("📅 Upcoming Events (Next %d days)\n", CLI.Cal.Days)
				fmt.Println(strings.Repeat("=", 60))
				
				for _, event := range events {
					timeStr := event.StartTime.Format("Mon, Jan 2 at 3:04 PM")
					if event.AllDay {
						timeStr = event.StartTime.Format("Mon, Jan 2 (All Day)")
					}
					
					fmt.Printf("\n🕒 %s\n", timeStr)
					fmt.Printf("📝 %s\n", event.Title)
					if event.CalendarName != "" {
						fmt.Printf("📅 Calendar: %s\n", event.CalendarName)
					}
					if event.Description != "" {
						fmt.Printf("📄 %s\n", event.Description)
					}
					if event.Location != "" {
						fmt.Printf("📍 %s\n", event.Location)
					}
				}
				fmt.Println()
			}
		}
	case "cal daemon":
		{
			// Run calendar notification daemon
			log.Printf("Starting calendar notification daemon")
			_ = io_helpers.Notify("Calendar Daemon Started", "Gmail Watcher")
			
			for _, calSrv := range calServices {
				calService := gcalendar.NewCalendarService(calSrv)
				
				// Run daemon in background for each calendar service
				go func(cs *gcalendar.CalendarService) {
					err := cs.RunCalendarDaemon(CLI.Cal.Daemon.CheckInterval, CLI.Cal.Daemon.NotifyBefore)
					if err != nil {
						log.Printf("Calendar daemon error: %v", err)
					}
				}(calService)
			}
			
			// Set up graceful shutdown
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			
			// Wait for shutdown signal
			<-sigCh
			log.Println("Calendar daemon shutdown signal received, stopping gracefully...")
			_ = io_helpers.Notify("Calendar Daemon Shutting Down", "Gmail Watcher")
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
		fmt.Printf("Make sure gmail readonlyscope and calendar readonly scope is available for the account\n")
		log.Fatalf("Unable to parse client secret file to config or obtain permissions\n")
	}
}
