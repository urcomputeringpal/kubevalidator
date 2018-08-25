package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
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

// Run starts a http server on the configured port
func (s *Server) Run(ctx context.Context) error {
	s.tr = &http.DefaultTransport

	itr, err := ghinstallation.NewAppsTransportKeyFromFile(*s.tr, s.AppID, s.PrivateKeyFile)
	if err != nil {
		return err
	}

	s.ctx = &ctx
	s.GitHubAppClient = github.NewClient(&http.Client{Transport: itr})

	http.HandleFunc("/webhook", s.handle)
	http.HandleFunc("/healthz", s.health)
	http.HandleFunc("/", s.redirect)
	log.Println("hi")
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil)
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(s.WebhookSecret))
	if err != nil {
		log.Println(err)
		return
	}
	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Println(err)
		return
	}

	ge := &GenericEvent{}
	err = json.Unmarshal(payload, &ge)
	if err != nil {
		log.Println(err)
		return
	}

	// TODO what happens if the event doesn't have an installation ID?
	itr, err := ghinstallation.NewKeyFromFile(*s.tr, s.AppID, int(ge.Installation.GetID()), s.PrivateKeyFile)
	if err != nil {
		log.Println(err)
		return
	}

	c := &Context{
		Event:  event,
		Ctx:    s.ctx,
		Github: github.NewClient(&http.Client{Transport: itr}),
	}

	// TODO Return a 500 if we don't make it through the complete CheckRun cycle
	c.Process()
	return
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	// TODO better health checks
	fmt.Fprintf(w, "hi")
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	// TODO automatically generate this redirect
	http.Redirect(w, r, "http://github.com/urcomputeringpal/kubevalidator", 301)
}
