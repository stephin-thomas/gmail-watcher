package common

type MailClient interface {
	Save() error
	Update() error
}
