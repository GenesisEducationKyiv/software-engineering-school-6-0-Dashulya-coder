package urlbuilder

import "fmt"

type URLBuilder interface {
	ConfirmURL(token string) string
	UnsubscribeURL(token string) string
}

type Builder struct {
	baseURL string
}

func New(baseURL string) *Builder {
	return &Builder{baseURL: baseURL}
}

func (b *Builder) ConfirmURL(token string) string {
	return fmt.Sprintf("%s/api/confirm/%s", b.baseURL, token)
}

func (b *Builder) UnsubscribeURL(token string) string {
	return fmt.Sprintf("%s/api/unsubscribe/%s", b.baseURL, token)
}

var _ URLBuilder = (*Builder)(nil)
