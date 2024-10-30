package main

var CLI struct {
	Login struct {
		AuthPort uint64 `default:"3000" help: "Redirect port url after authentication"`
	} `cmd:"" help:"Login to google"`
	Gmail struct {
		MaxRetries uint8 `default:"3" help: "Max retries to connect to server"`
		List       struct {
			ListLen  uint8 `default:"15" help: "Length of total emails to be displayed from each accounts"`
			Parallel bool  `help: "Retrieve messages in parallel"`
		} `cmd:"" help:"List all retrieved email"`
		Daemon struct {
			Retries          uint8 `default:"3" help: "Max retries before timeout"`
			MaxNotifications uint8 `default:"7" help: "Max number of notifications of updated email shown"`
		} `cmd:"" help:"Run a gmail notification daemon"`
	} `cmd:"" help:"gmail"`
	Cal struct {
		MaxResults int64 `default:"10" help: "Max Events should be shown from each calendar"`
	} `cmd:"" help:"Show calendar events"`
}
