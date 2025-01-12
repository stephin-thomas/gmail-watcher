package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gmail-watcher/exports"
	"github.com/gmail-watcher/gmail_client"
	"github.com/gmail-watcher/io_helpers"
	"golang.org/x/oauth2"
)

type UserConfig struct {
	UserToken *oauth2.Token
	gmail_client.GmailUserConfig
}
type AppConfig struct {
	UserConfigs *[]UserConfig
}

func LoadAppConfig() (*AppConfig, error) {
	var app_config AppConfig
	err := io_helpers.DeserializeFromFile(exports.APP_CONFIG, &app_config)
	if err != nil {
		return nil, fmt.Errorf("error loading appConfig %w", err)
	}
	return &app_config, nil
}

func removeLoginIndices(slice *[]UserConfig, indices []int) *[]UserConfig {
	// Sort indices in descending order to avoid index shifts during removal
	sort.Sort(sort.Reverse(sort.IntSlice(indices)))

	for _, index := range indices {
		// Validate index within slice bounds
		if index < 0 || index >= len(*slice) {
			continue
		}

		// Remove element at index
		*slice = append((*slice)[:index], (*slice)[index+1:]...)
	}

	return slice
}
func isGmailSubCommand(cli_cmd string) bool {
	subCommand := strings.Fields(cli_cmd)
	isSubCommand := len(subCommand) > 1
	if isSubCommand {
		return subCommand[0] == "gmail"
	}
	return false
}
func isCalSubCommand(cli_cmd string) bool {
	subCommand := strings.Fields(cli_cmd)
	isSubCommand := len(subCommand) > 1
	if isSubCommand {
		return subCommand[0] == "cal"
	}
	return false
}
