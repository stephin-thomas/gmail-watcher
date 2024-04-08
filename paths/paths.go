package paths

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

func get_config_folder() string {
	var CONFIG_FOLDER string = filepath.Join(xdg.ConfigHome, "gmail_watcher")
	return CONFIG_FOLDER
}

var CONFIG_FOLDER string = get_config_folder()
var CREDENTIALS_FILE = filepath.Join(CONFIG_FOLDER, "credentials.json")
var LOGIN_TOKENS_LIST_FILE = filepath.Join(CONFIG_FOLDER, "login_tokens.json")
var PORT int64 = 5000
var NOTIFICATION_ICON = filepath.Join(CONFIG_FOLDER, "assets/notification.png")
var LOG_FILE_PATH = filepath.Join(CONFIG_FOLDER, "gmail-watcher.log")
var ASSETS_SOURCE_PATH = "assets/notification.png"
var ASSETS_PATH = filepath.Join(CONFIG_FOLDER, ASSETS_SOURCE_PATH)
