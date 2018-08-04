package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
)

type kubevalidator struct {
	Port             int
	WebhookSecretKey string
	GitHubAppKeyFile string
	GitHubAppID      int
	GitHubAppClient  *github.Client
	tr               http.RoundTripper
	ctx              *context.Context
}

func (kv *kubevalidator) handle(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(kv.WebhookSecretKey))
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

func (kv *kubevalidator) health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "cool")
}

func runWithContext(ctx context.Context) error {
	kv := kubevalidator{}
	flag.IntVar(&kv.Port, "port", 8080, "port to listen on")
	flag.StringVar(&kv.WebhookSecretKey, "webhook-secret", "", "webhook secret")
	flag.StringVar(&kv.GitHubAppKeyFile, "github-app-key-file", "", "path to key file")
	flag.IntVar(&kv.GitHubAppID, "github-app-id", -1, "app ID")
	flag.Parse()
	if len(flag.Args()) > 0 {
		fmt.Printf("Unparsed arguments provided:\n\n%+v\n\n", flag.Args())
		flag.Usage()
		os.Exit(2)
	}

	itr, err := ghinstallation.NewAppsTransportKeyFromFile(kv.tr, kv.GitHubAppID, kv.GitHubAppKeyFile)
	if err != nil {
		return err
	}

	kv.ctx = &ctx
	kv.tr = http.DefaultTransport
	kv.GitHubAppClient = github.NewClient(&http.Client{Transport: itr})

	http.HandleFunc("/webhook", kv.handle)
	http.HandleFunc("/health", kv.health)
	log.Println("hi")
	return http.ListenAndServe(fmt.Sprintf(":%d", kv.Port), nil)
}

func cancelOnInterrupt(ctx context.Context, f context.CancelFunc) {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-term:
			log.Println("Received SIGTERM, exiting gracefully...")
			f()
			os.Exit(0)
		case <-ctx.Done():
			os.Exit(0)
		}
	}
}

func run() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go cancelOnInterrupt(ctx, cancelFunc)

	return runWithContext(ctx)
}

func main() {
	if err := run(); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		panic(err)
	}
}
