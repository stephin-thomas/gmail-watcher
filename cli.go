package main

var CLI struct {
	Login struct{} `cmd:"" help:"Login to google"`
	Gmail struct {
		MaxRetries uint8 `default:"3" help: "Max retries to connect to server"`
		List       struct {
			ListLen uint8 `default:"15" help: "Length of total emails to be displayed from each accounts"`
		} `cmd:"" help:"List all retrieved email"`
		Daemon struct{} `cmd:"" help:"Run a gmail notification daemon"`
	} `cmd:"" help:"gmail"`
	Cal struct{} `cmd:"" help:"Show calendar events"`
}
