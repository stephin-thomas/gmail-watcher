package main

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
	Gmail struct {
		MaxRetries uint8 `default:"2" help: "Max retries to connect to server"`
		List       struct {
			ListLen  uint8 `default:"15" help: "Length of total emails to be displayed from each accounts"`
			Parallel bool  `help: "Retrieve messages in parallel"`
		} `cmd:"" help:"List all retrieved email"`
		Daemon struct {
			Retries          uint8 `default:"7" help: "Max retries before timeout"`
			MaxNotifications uint8 `default:"7" help: "Max number of notifications of updated email shown"`
		} `cmd:"" help:"Run a gmail notification daemon"`
	} `cmd:"" help:"gmail"`
	Cal struct {
		MaxResults int64 `default:"10" help: "Max Events should be shown from each calendar"`
	} `cmd:"" help:"Show calendar events"`
}
