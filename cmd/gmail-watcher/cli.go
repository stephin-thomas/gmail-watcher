// Package main provides the command-line interface for Gmail Watcher.
// Gmail Watcher is a desktop notification service for Gmail and Google Calendar.
package main

// CLI defines the complete command-line interface structure using Kong framework.
// It supports unified daemon mode, individual service daemons, and utility commands.
var CLI struct {
	Login struct {
		Auth struct {
			AuthPort uint64 `default:"3000" help: "Redirect port url after authentication"`
		} `cmd:"" help: "Add login"`
		ListUsers struct{} `cmd:"" help: "Display all logged in users"`
		Delete    struct {
			Index []int `type:"login index" help "Delete the login at the index"`
			All   bool  `help:"delete all logins"`
		} `cmd"" help:"Delete logins"`
	} `cmd:"" help:"Login to google"`
	
	// Unified daemon command - Primary way to run Gmail Watcher
	// Monitors both Gmail and Calendar with configurable service enablement
	Daemon struct {
		// Gmail notification settings
		GmailEnabled         bool  `default:"true" help: "Enable Gmail notifications (default: true)"`
		GmailRetries         uint8 `default:"7" help: "Max retries before Gmail timeout (default: 7)"`
		GmailMaxNotifications uint8 `default:"7" help: "Max number of Gmail notifications per cycle (default: 7)"`
		
		// Calendar notification settings  
		CalendarEnabled       bool  `default:"true" help: "Enable Calendar notifications (default: true)"`
		CalendarCheckInterval int   `default:"5" help: "Calendar check interval in minutes (default: 5)"`
		CalendarNotifyBefore  []int `default:"15,5,0" help: "Minutes before events to notify: 15,5,0 means 15min, 5min, and at event start"`
		CalendarMaxNotifications int `default:"10" help: "Maximum calendar notifications per check cycle (default: 10)"`
	} `cmd:"" help:"Run unified notification daemon for Gmail and Calendar"`
	
	Gmail struct {
		MaxRetries uint8 `default:"2" help: "Max retries to connect to server"`
		List       struct {
			ListLen  uint8 `default:"15" help: "Length of total emails to be displayed from each accounts"`
			Parallel bool  `help: "Retrieve messages in parallel"`
		} `cmd:"" help:"List all retrieved email"`
		Daemon struct {
			Retries          uint8 `default:"7" help: "Max retries before timeout"`
			MaxNotifications uint8 `default:"7" help: "Max number of notifications of updated email shown"`
		} `cmd:"" help:"Run Gmail-only notification daemon"`
	} `cmd:"" help:"Gmail operations"`
	
	Cal struct {
		MaxResults int64 `default:"10" help: "Max Events should be shown from each calendar"`
		Days       int   `default:"7" help: "Number of days ahead to show events"`
		Daemon     struct {
			CheckInterval    int   `default:"5" help: "Check interval in minutes for upcoming events"`
			NotifyBefore     []int `default:"15,5,0" help: "Minutes before event to send notifications"`
			MaxNotifications int   `default:"10" help: "Maximum notifications per check"`
		} `cmd:"" help:"Run Calendar-only notification daemon"`
	} `cmd:"" help:"Calendar events"`
}
