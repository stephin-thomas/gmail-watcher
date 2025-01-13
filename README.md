# gmail-watcher
This is a gmail email notifier for linux written in go.
Follow the steps '**Enable the API**' and '**Authorize credentials for a desktop application**' from the following page

*Note:- Ignore all other steps mentioned in the page*

https://developers.google.com/gmail/api/quickstart/go



Rename the downloaded file to credentials.json and copy it to
```
~/.config/gmail_watcher
```
Run `gmail-watcher --help` for more information 

# Building
```
mkdir ./build
go build -o ./build/ ./cmd/gmail-watcher
```
