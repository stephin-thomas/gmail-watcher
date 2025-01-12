package exports

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

func get_config_folder() string {
	var CONFIG_FOLDER string = filepath.Join(xdg.ConfigHome, "gmail_watcher")
	return CONFIG_FOLDER
}
func get_data_folder() string {
	DATA_FOLDER := filepath.Join(xdg.DataHome, "gmail_watcher")
	return DATA_FOLDER
}

var CONFIG_FOLDER string = get_config_folder()
var CREDENTIALS_FILE = filepath.Join(CONFIG_FOLDER, "credentials.json")
var APP_CONFIG = filepath.Join(CONFIG_FOLDER, "config.json")
var DATA_FOLDER = get_data_folder()

// var PORT uint64 = 5000
var LOG_FILE_PATH = filepath.Join(DATA_FOLDER, "gmail-watcher.log")
var ASSETS_SOURCE_PATH = "assets/"
var ASSETS_PATH = filepath.Join(DATA_FOLDER, ASSETS_SOURCE_PATH)
var NOTIFICATION_ICON = filepath.Join(ASSETS_PATH, "notification.png")
