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
	log.Printf("Gmail services count: %d", len(*clientSrvs))
	
	if len(*clientSrvs) == 0 {
		log.Println("ERROR: No Gmail services provided to daemon!")
		return fmt.Errorf("no gmail services provided")
	}
	
	// Log each service for debugging
	for i, srv := range *clientSrvs {
		log.Printf("Gmail service %d: %s (DB: %s)", i+1, srv.EmailID, srv.DBPath)
	}
	
	// Notify systemd that the service is ready (for systemd integration)
	_, _ = daemon.SdNotify(true, daemon.SdNotifyReady)
	
	// Main daemon loop - runs indefinitely until error or shutdown
	for {
		var retry uint8 = 0
		log.Printf("Starting new daemon cycle with %d Gmail services", len(*clientSrvs))
		
		// Process each Gmail service account
		for i, clientSrv := range *clientSrvs {
			log.Printf("Processing Gmail account %d: %s", i+1, clientSrv.EmailID)
			
			// Check for new messages since last check
			updatedMsgList, err1 := clientSrv.UpdateMsgIDs()
			if err1 != nil {
				retry++
				log.Printf("ERROR: Failed to update message IDs for %s: %v", clientSrv.EmailID, err1)
				log.Println("Error getting msg list sleeping and trying again", err1)
				
				// Show error notification to user
				errorTitle := fmt.Sprintf("Gmail Error - %s", clientSrv.EmailID)
				errorMsg := fmt.Sprintf("Failed to fetch emails: %v", err1)
				_ = io_helpers.Notify(errorMsg, errorTitle)
				
				time.Sleep(DefaultRetryInterval)
				handleRetries(&retry, maxRetries, err1)
				continue // Skip to next account on error
			}
			
			log.Printf("UpdateMsgIDs returned %d new messages for %s", len(updatedMsgList), clientSrv.EmailID)

		// Process new messages if any were found
		if len(updatedMsgList) > 0 {
			// Save the updated message IDs to persistent storage
			err := clientSrv.Save()
			if err != nil {
				errorTitle := fmt.Sprintf("Database Error - %s", clientSrv.EmailID)
				errorMsg := fmt.Sprintf("Failed to save message database: %v", err)
				_ = io_helpers.Notify(errorMsg, errorTitle)
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
				log.Printf("ERROR: Failed to fetch message details for %s: %v", clientSrv.EmailID, err2)
				
				// Show error notification to user
				errorTitle := fmt.Sprintf("Gmail Fetch Error - %s", clientSrv.EmailID)
				errorMsg := fmt.Sprintf("Failed to fetch message details: %v", err2)
				_ = io_helpers.Notify(errorMsg, errorTitle)
				
				time.Sleep(DefaultRetryInterval)
				log.Println("error getting messages for the updated msg ids. sleeping and trying again", err2)
				continue
			} else {
				// Reset retry counter on successful operation
				retry = 0
				
				// Send desktop notification for each new message
				for _, msg := range *msgs {
					sender := gmail_client.GetSender(msg)
					// Include receiving email account in notification title
					notificationTitle := fmt.Sprintf("%s → %s", sender, clientSrv.EmailID)
					_ = io_helpers.Notify(msg.Snippet, notificationTitle)
				}
			}
		}
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
		errMsg := "Unable to retrieve emails. Please check your internet connection and account permissions."
		errTitle := "Gmail Watcher - Connection Issue"
		_ = io_helpers.Notify(errMsg, errTitle)
	}
	
	// Sleep before retry attempt
	time.Sleep(DefaultRetryInterval)
	
	// Handle maximum retries reached
	if *retry == maxRetries {
		errMsg := fmt.Sprintf("Max retries (%d) reached. Gmail Watcher will pause for 5 minutes before trying again.", maxRetries)
		errTitle := "Gmail Watcher - Service Paused"
		_ = io_helpers.Notify(errMsg, errTitle)
		log.Printf("Max retries reached, pausing for 5 minutes")
		
		// Extended sleep to prevent rapid restart loops
		time.Sleep(5 * time.Minute)
	}
}
