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
	log.Printf("Token file name generated (count: %d)", len(*tokFiles))
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

// Get sender from the gmail.Message headers
func GetSender(msg *gmail.Message) string {
	for _, header := range msg.Payload.Headers {
		if header.Name == "From" {
			return parseFromHeader(header.Value)
		}
	}
	return "<No From Address>"
}

// parseFromHeader extracts the display name from an email "From" header
// Examples:
// "John Doe <john@example.com>" -> "John Doe"
// "john@example.com" -> "john@example.com"
func parseFromHeader(fromHeader string) string {
	// Look for pattern: "Display Name <email@domain.com>"
	if len(fromHeader) > 0 {
		// Check if there's a display name in quotes
		if fromHeader[0] == '"' {
			// Find closing quote
			endQuote := 1
			for endQuote < len(fromHeader) && fromHeader[endQuote] != '"' {
				endQuote++
			}
			if endQuote < len(fromHeader) {
				return fromHeader[1:endQuote]
			}
		}

		// Check for unquoted display name before angle bracket
		angleIndex := -1
		for i, char := range fromHeader {
			if char == '<' {
				angleIndex = i
				break
			}
		}

		if angleIndex > 0 {
			// Extract display name, trim spaces
			displayName := fromHeader[:angleIndex]
			// Remove leading/trailing spaces
			start := 0
			end := len(displayName)
			for start < end && displayName[start] == ' ' {
				start++
			}
			for end > start && displayName[end-1] == ' ' {
				end--
			}
			if end > start {
				return displayName[start:end]
			}
		}
	}

	// If no display name found, return the original header (just email)
	return fromHeader
}
