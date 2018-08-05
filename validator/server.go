package validator

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
)

// Validator contains the logic to dispatch PRs to kubeval
type Validator struct {
	Port            int
	WebhookSecret   string
	PrivateKeyFile  string
	AppID           int
	GitHubAppClient *github.Client
	tr              http.RoundTripper
	ctx             *context.Context
}

// Run starts a http server on the configured port
func (v *Validator) Run(ctx context.Context) error {
	v.tr = http.DefaultTransport

	itr, err := ghinstallation.NewAppsTransportKeyFromFile(v.tr, v.AppID, v.PrivateKeyFile)
	if err != nil {
		return err
	}

	v.ctx = &ctx
	v.GitHubAppClient = github.NewClient(&http.Client{Transport: itr})

	http.HandleFunc("/webhook", v.handle)
	http.HandleFunc("/", v.health)
	log.Println("hi")
	return http.ListenAndServe(fmt.Sprintf(":%d", v.Port), nil)
}

func (v *Validator) handle(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(v.WebhookSecret))
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

	switch e := event.(type) {
	case *github.CheckSuiteEvent:
		log.Printf("received %s\n", event)
		return
	default:
		log.Printf("ignoring %s\n", e)
		return
	}
}

func (v *Validator) health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hi")
}
