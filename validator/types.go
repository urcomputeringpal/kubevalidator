package validator

import (
	"context"
	"net/http"

	"github.com/google/go-github/github"
)

// Server contains the logic to process webhooks, kinda like probot
type Server struct {
	Port            int
	WebhookSecret   string
	PrivateKeyFile  string
	AppID           int
	GitHubAppClient *github.Client
	tr              *http.RoundTripper
	ctx             *context.Context
}

// GenericEvent contains just enough inforamation about webhook to handle
// authentication
type GenericEvent struct {
	// Repo         *github.Repository   `json:"repository,omitempty"`
	// Org          *github.Organization `json:"organization,omitempty"`
	// Sender       *github.User         `json:"sender,omitempty"`
	Installation *github.Installation `json:"installation,omitempty"`
}

// Context contains an event payload an a configured client
type Context struct {
	event  interface{}
	github *github.Client
	ctx    *context.Context
}
