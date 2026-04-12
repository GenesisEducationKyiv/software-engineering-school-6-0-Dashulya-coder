package mailer

type Mailer interface {
	SendConfirmation(email, confirmLink string) error
	SendNewRelease(email, repo, tag, releaseURL, unsubscribeLink string) error
}
