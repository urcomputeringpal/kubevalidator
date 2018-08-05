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
	http.HandleFunc("/", s.health)
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
		event:  event,
		ctx:    s.ctx,
		github: github.NewClient(&http.Client{Transport: itr}),
	}

	c.Process()
	return
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hi")
}
