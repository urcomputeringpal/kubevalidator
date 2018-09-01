package validator

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-github/github"
)

func TestPullRequestTestingHappyPath(t *testing.T) {
	prEvent := &github.PullRequestEvent{
		Action: github.String("opened"),
		PullRequest: &github.PullRequest{
			Head: &github.PullRequestBranch{
				Ref: github.String("b"),
			},
		},
		Repo: &github.Repository{
			Owner: &github.User{
				Login: github.String("o"),
			},
			Name: github.String("r"),
		},
	}
	client, mux, _, teardown := setup()
	ctx := context.Background()
	context := &Context{
		Ctx:    &ctx,
		Event:  prEvent,
		Github: client,
		AppID:  github.Int(1),
	}
	defer teardown()
	mux.HandleFunc("/repos/o/r/commits/b/check-suites", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{"app_id": "1"})
		fmt.Fprintf(w, `{
			"total_count": 1,
			"check_suites": [
				{
					"id": 5,
					"pull_requests": [
					]
				}
			]
		}`)
	})
	mux.HandleFunc("/repos/o/r/check-suites/5/rerequest", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testBody(t, r, "")
	})
	processed := context.Process()
	if !processed {
		t.Error("PR event was never processed")
	}
	return
}

func TestPullRequestTestingTooManyCheckSuites(t *testing.T) {
	prEvent := &github.PullRequestEvent{
		Action: github.String("opened"),
		PullRequest: &github.PullRequest{
			Head: &github.PullRequestBranch{
				Ref: github.String("b"),
			},
		},
		Repo: &github.Repository{
			Owner: &github.User{
				Login: github.String("o"),
			},
			Name: github.String("r"),
		},
	}
	client, mux, _, teardown := setup()
	ctx := context.Background()
	context := &Context{
		Ctx:    &ctx,
		Event:  prEvent,
		Github: client,
		AppID:  github.Int(1),
	}
	defer teardown()
	mux.HandleFunc("/repos/o/r/commits/b/check-suites", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{"app_id": "1"})
		fmt.Fprintf(w, `{
			"total_count": 2,
			"check_suites": [
				{
					"id": 5,
					"pull_requests": [
					]
				},
				{
					"id": 6,
					"pull_requests": [
					]
				}

			]
		}`)
	})
	processed := context.Process()
	if processed {
		t.Error("PR event expected to be skipped")
	}
	return
}
