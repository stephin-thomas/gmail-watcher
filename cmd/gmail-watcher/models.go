package main

import (
	"fmt"
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

func (app_config AppConfig) Save() error {
	err := io_helpers.SerializeNsave(app_config, exports.APP_CONFIG)
	if err != nil {
		return fmt.Errorf("Unable to save App configurations %w", err)
	}
	return nil
}

func LoadAppConfig() (*AppConfig, error) {
	var app_config AppConfig
	err := io_helpers.DeserializeFromFile(exports.APP_CONFIG, &app_config)
	if err != nil {
		return nil, fmt.Errorf("error loading appConfig %w", err)
	}
	return &app_config, nil
}

func deleteSliceIndex(slice *[]UserConfig, index int) {
	*slice = append((*slice)[:index], (*slice)[index+1:]...)
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
