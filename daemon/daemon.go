// Package daemon provides the core Gmail notification daemon functionality.
// It continuously monitors Gmail accounts for new messages and sends desktop notifications.
package daemon

import (
	"fmt"
	"log"
	"time"

	"github.com/coreos/go-systemd/daemon"
	"github.com/gmail-watcher/gmail_client"

	"github.com/gmail-watcher/io_helpers"
)

const (
	// DefaultMaxMessagesPerAccount is the default maximum number of messages to process per account
	DefaultMaxMessagesPerAccount = 8
	// DefaultSleepInterval is the default sleep interval between daemon cycles
	DefaultSleepInterval = 30 * time.Second
	// DefaultRetryInterval is the default retry interval on errors
	DefaultRetryInterval = 10 * time.Second
)

// RunDaemon starts the Gmail notification daemon that continuously monitors Gmail accounts.
// It processes multiple Gmail services concurrently and handles retries on failures.
//
// Parameters:
//   - maxRetries: Maximum number of retries before giving up on error conditions
//   - maxNotifications: Maximum number of notifications to show per cycle (currently unused)
//   - clientSrvs: Slice of Gmail service clients to monitor
//
// Returns:
//   - error: Any fatal error that caused the daemon to stop
func RunDaemon(maxRetries uint8, maxNotifications uint8, clientSrvs *[]*gmail_client.GmailService) error {
	log.Println("Daemon started")
	
	// Notify systemd that the service is ready (for systemd integration)
	_, _ = daemon.SdNotify(true, daemon.SdNotifyReady)
	
	// Main daemon loop - runs indefinitely until error or shutdown
	for {
		var retry uint8 = 0
		
		// Process each Gmail service account
		i := 0
		for i < len(*clientSrvs) {
			clientSrv := (*clientSrvs)[i]
			
			// Check for new messages since last check
			updatedMsgList, err1 := clientSrv.UpdateMsgIDs()
			if err1 != nil {
				retry++
				log.Println("Error getting msg list sleeping and trying again", err1)
				time.Sleep(DefaultRetryInterval)
				handleRetries(&retry, maxRetries, err1)
				continue
			}

		// Process new messages if any were found
		if len(updatedMsgList) > 0 {
			// Save the updated message IDs to persistent storage
			err := clientSrv.Save()
			if err != nil {
				_ = io_helpers.Notify("Error saving database", "Error!")
				log.Printf("error saving db database %v\n %v", clientSrv.DBPath, err)
				return fmt.Errorf("error saving db database %v\n %w", clientSrv.DBPath, err)
			}
			
			// Limit the number of messages processed to avoid overwhelming the user
			log.Printf("Total msgs from google: %d\n Using only: %d", len(updatedMsgList), DefaultMaxMessagesPerAccount)
			if len(updatedMsgList) > DefaultMaxMessagesPerAccount {
				updatedMsgList = updatedMsgList[:DefaultMaxMessagesPerAccount]
			}
			
			// Fetch full message details for the new messages
			msgs, err2 := clientSrv.FetchAllMail(updatedMsgList)

			if err2 != nil {
				retry++
				time.Sleep(DefaultRetryInterval)
				log.Println("error getting messages for the updated msg ids. sleeping and trying again", err2)
				continue
			} else {
				// Reset retry counter on successful operation
				retry = 0
				
				// Send desktop notification for each new message
				for _, msg := range *msgs {
					sender := gmail_client.GetSender(msg)
					_ = io_helpers.Notify(msg.Snippet, sender)
				}
			}
		}
			// Move to next Gmail account
			i++
		}
		
		// Sleep between daemon cycles to avoid excessive API usage
		log.Printf("Sleeping for %v", DefaultSleepInterval)
		if retry == maxRetries {
			_ = io_helpers.Notify("Unable to retrieve email", "Gmail Watcher Error!")
		}
		time.Sleep(DefaultSleepInterval)
	}

	// Notify systemd that the service is stopping (unreachable in current implementation)
	_, _ = daemon.SdNotify(true, daemon.SdNotifyStopping)
	return nil
}

// handleRetries manages retry logic and user notifications for daemon errors.
// It sends escalating notifications to the user and implements exponential backoff.
//
// Parameters:
//   - retry: Pointer to current retry count (modified by this function)
//   - maxRetries: Maximum allowed retries before giving up
//   - err: The error that triggered the retry
func handleRetries(retry *uint8, maxRetries uint8, err error) {
	log.Println("Error connecting", err)
	
	// Send initial warning notification to user
	if *retry == 1 {
		errMsg := "Unable retrieve emails please check your internet connection"
		errTitle := "Error"
		_ = io_helpers.Notify(errMsg, errTitle)
	}
	
	// Sleep before retry attempt
	time.Sleep(DefaultRetryInterval)
	
	// Handle maximum retries reached
	if *retry == maxRetries {
		errMsg := "Shutting down Gmail Watcher due to errors"
		errTitle := "Error"
		_ = io_helpers.Notify(errMsg, errTitle)
		log.Printf("Shutting down gmail watcher due to errors")
		
		// Extended sleep to prevent rapid restart loops
		time.Sleep(5 * time.Minute)
	}
}
