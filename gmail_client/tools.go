package gmail_client

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/gmail-watcher/exports"
	"github.com/gmail-watcher/io_helpers"
	"github.com/google/uuid"
	"google.golang.org/api/gmail/v1"
)

func AddRandomTokenPath(tokFiles *[]string) *string {
	token_file_name := fmt.Sprintf("token_%s.json", uuid.NewString())
	token_file_path := path.Join(exports.CONFIG_FOLDER, token_file_name)
	*tokFiles = append(*tokFiles, token_file_path)
	log.Printf("Token file name generated %v", tokFiles)
	return &token_file_path
}

func CreateIDList(records *[]*gmail.Message) *map[string]struct{} {
	id_list := map[string]struct{}{}
	for _, msg := range *records {
		id_list[msg.Id] = struct{}{}
		// id_list = append(id_list, msg.Id) // note the = instead of :=
	}
	return &id_list
}

func AddToken(tokFiles *[]string) (*string, error) {
	log.Println("Adding new token")
	tok_file_name := AddRandomTokenPath(tokFiles)
	log.Println("Added token file to:-", tok_file_name)
	err := io_helpers.SerializeNsave(tokFiles, exports.APP_CONFIG)
	if err != nil {
		return nil, fmt.Errorf("error adding tokens:- %w", err)
	}
	return tok_file_name, nil
}

func LoadIDList(file_name string) (map[string]struct{}, error) {
	var id_list map[string]struct{}
	var json_file []byte
	var err error
	log.Println("Loading IDs from", file_name)
	if !io_helpers.FileExists(file_name) {
		// id_list_nul := make(map[string]struct{}, 0)
		return nil, fmt.Errorf("id list file don't exist %w", err)
	}
	json_file, err = os.ReadFile(file_name)
	if err != nil {
		return nil, fmt.Errorf("error reading ID list %w", err)
	}

	err = json.Unmarshal(json_file, &id_list)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling ID list %w", err)
	}
	return id_list, nil
}

// func token_expired(token_expiry *time.Time) bool {
// 	cur_time := time.Now()
// 	if cur_time.Sub(*token_expiry).Seconds() >= 0 {
// 		return true
// 	} else {
// 		return false
// 	}
// }

// Get sender from the gmail.Message headers
func GetSender(msg *gmail.Message) string {
	for _, header := range msg.Payload.Headers {
		if header.Name == "From" {
			return header.Value
		}
	}
	return "<No From Address>"
}
