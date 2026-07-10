package mailer

type ConfirmationSender interface {
	SendConfirmation(email, confirmLink string) error
}

type ReleaseSender interface {
	SendNewRelease(email, repo, tag, releaseURL, unsubscribeLink string) error
}

type Mailer interface {
	ConfirmationSender
	ReleaseSender
}
