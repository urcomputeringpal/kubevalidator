package validator

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const (
	checkRunName           = "kubevalidator"
	initialCheckRunSummary = "Validating..."
	noMatchingFiles        = "No files to validate"
	configPath             = ".github/kubevalidator.yaml"
)

// createInitialCheckRun contains the logic which sets the title and summary
// of the check
func (c *Context) createInitialCheckRun(e *github.CheckSuiteEvent) error {
	checkRunOpt := github.CreateCheckRunOptions{
		Name:       checkRunName,
		HeadBranch: e.CheckSuite.GetHeadBranch(),
		HeadSHA:    e.CheckSuite.GetHeadSHA(),
		Status:     github.String("in_progress"),
		StartedAt:  &github.Timestamp{Time: time.Now()},
		Output: &github.CheckRunOutput{
			Title:   github.String(initialCheckRunSummary),
			Summary: github.String(initialCheckRunSummary),
		},
	}

	_, _, err := c.Github.Checks.CreateCheckRun(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), checkRunOpt)
	if err != nil {
		log.Println(errors.Wrap(err, "Couldn't create check run"))
		return err
	}
	return nil
}

func (c *Context) createConfigMissingCheckRun(startedAt *time.Time, e *github.CheckSuiteEvent) error {
	checkRunOpt := github.CreateCheckRunOptions{
		Name:        checkRunName,
		HeadBranch:  e.CheckSuite.GetHeadBranch(),
		HeadSHA:     e.CheckSuite.GetHeadSHA(),
		Status:      github.String("completed"),
		Conclusion:  github.String("neutral"),
		StartedAt:   &github.Timestamp{Time: *startedAt},
		CompletedAt: &github.Timestamp{Time: time.Now()},
		Output: &github.CheckRunOutput{
			Title:       github.String("No configuration"),
			Summary:     github.String(fmt.Sprintf("kubevalidator needs a tiny bit of configuration to know where to find the Kubernetes YAML in your Repository.\n\n1. Check out the [documentation and examples](https://github.com/urcomputeringpal/kubevalidator#configuration).\n1. Add your configuration to [`.github/kubevalidator.yaml`](https://github.com/%s/%s/new/%s?filename=.github/kubevalidator.yaml)\n1. Profit???", e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), e.CheckSuite.GetHeadBranch())),
			Annotations: nil,
		},
	}

	_, _, err := c.Github.Checks.CreateCheckRun(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), checkRunOpt)
	if err != nil {
		log.Println(errors.Wrap(err, "Couldn't create check run"))
		return err
	}
	return nil
}

func (c *Context) createConfigInvalidCheckRun(startedAt *time.Time, e *github.CheckSuiteEvent, annotations []*github.CheckRunAnnotation) error {
	configURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), e.CheckSuite.GetHeadBranch(), configPath)
	checkRunOpt := github.CreateCheckRunOptions{
		Name:        checkRunName,
		HeadBranch:  e.CheckSuite.GetHeadBranch(),
		HeadSHA:     e.CheckSuite.GetHeadSHA(),
		Status:      github.String("completed"),
		Conclusion:  github.String("failure"),
		StartedAt:   &github.Timestamp{Time: *startedAt},
		CompletedAt: &github.Timestamp{Time: time.Now()},
		Output: &github.CheckRunOutput{
			Title:       github.String("Configuration invalid"),
			Summary:     github.String(fmt.Sprintf("Check out the [documentation and examples](https://github.com/urcomputeringpal/kubevalidator#configuration) and [update your configuration to match](%v). Please do [reach out](https://github.com/urcomputeringpal/kubevalidator/issues/new/choose) if you're having trouble or think you've have found a bug!", configURL)),
			Annotations: annotations,
		},
	}

	_, _, err := c.Github.Checks.CreateCheckRun(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), checkRunOpt)
	if err != nil {
		log.Println(errors.Wrap(err, "Couldn't create check run"))
		return err
	}
	return nil
}

// createFinalCheckRun concludes the check run
func (c *Context) createFinalCheckRun(startedAt *time.Time, e *github.CheckSuiteEvent, candidates Candidates, annotations []*github.CheckRunAnnotation) error {
	var checkRunConclusion string
	var checkRunText string
	var checkRunSummary string
	numFiles := len(candidates)
	if numFiles == 0 {
		checkRunConclusion = "neutral"
		checkRunText = noMatchingFiles
		configURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), e.CheckSuite.GetHeadBranch(), configPath)
		checkRunSummary = fmt.Sprintf("None of the files changed on this Pull Request matched the configuration in [`%s`](%s). Please do [reach out](https://github.com/urcomputeringpal/kubevalidator/issues/new/choose) if you're having trouble or think you've have found a bug!", configPath, configURL)
	} else {
		// MVP pluralization
		filesString := "files"
		errorsString := "errors"

		if numFiles == 1 {
			filesString = "file"
		}

		if len(annotations) == 1 {
			errorsString = "error"
		}

		if len(annotations) > 0 {
			checkRunConclusion = "failure"
		} else {
			checkRunConclusion = "success"
		}
		checkRunText = fmt.Sprintf("%d %s checked, %d %s", numFiles, filesString, len(annotations), errorsString)

		var list []string
		for _, c := range candidates {
			list = append(list, c.MarkdownListItem())
		}
		checkRunSummary = strings.Join(list, "\n")
	}

	checkRunOpt := github.CreateCheckRunOptions{
		Name:        checkRunName,
		HeadBranch:  e.CheckSuite.GetHeadBranch(),
		HeadSHA:     e.CheckSuite.GetHeadSHA(),
		Status:      github.String("completed"),
		Conclusion:  &checkRunConclusion,
		StartedAt:   &github.Timestamp{Time: *startedAt},
		CompletedAt: &github.Timestamp{Time: time.Now()},
		Output: &github.CheckRunOutput{
			Title:       &checkRunText,
			Summary:     &checkRunSummary,
			Annotations: annotations,
		},
	}

	_, _, err := c.Github.Checks.CreateCheckRun(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), checkRunOpt)
	if err != nil {
		log.Println(errors.Wrap(err, "Couldn't create check run"))
		return err
	}
	return nil
}

func (c *Context) bytesForFilename(e *github.CheckSuiteEvent, f string) (*[]byte, error) {
	fileToValidate, _, _, err := c.Github.Repositories.GetContents(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), f, &github.RepositoryContentGetOptions{
		Ref: e.CheckSuite.GetHeadSHA(),
	})
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Couldn't load %s", f))
	}

	contentToValidate, err := fileToValidate.GetContent()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Couldn't load contents of %s", f))
	}

	b := []byte(contentToValidate)
	return &b, nil
}

func (c *Context) kubeValidatorConfigOrAnnotation(e *github.CheckSuiteEvent) (*KubeValidatorConfig, *github.CheckRunAnnotation, error) {
	config := &KubeValidatorConfig{}
	// TODO also support .github/kubevalidator.yml
	configBlobHRef := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), e.CheckSuite.GetHeadSHA(), configPath)
	configBytes, err := c.bytesForFilename(e, configPath)
	if err != nil {
		return nil, nil, err
	}
	if configBytes != nil {
		err := yaml.Unmarshal(*configBytes, config)
		if err != nil {
			return nil, &github.CheckRunAnnotation{
				Path:            github.String(configPath),
				BlobHRef:        &configBlobHRef,
				StartLine:       github.Int(1),
				EndLine:         github.Int(1),
				AnnotationLevel: github.String("failure"),
				Title:           github.String("Unmarshaling error"),
				Message:         github.String(fmt.Sprintf("%+v", err)),
			}, nil
		}
		if !config.Valid() {
			return nil, &github.CheckRunAnnotation{
				Path:            github.String(configPath),
				BlobHRef:        &configBlobHRef,
				StartLine:       github.Int(1),
				EndLine:         github.Int(1),
				AnnotationLevel: github.String("failure"),
				Message:         github.String("Schema validation error"),
			}, nil
		}
	}
	return config, nil, nil
}

func (c *Context) changedFileList(e *github.CheckSuiteEvent) ([]*github.CommitFile, error) {
	var prFiles []*github.CommitFile
	for _, pr := range e.CheckSuite.PullRequests {
		files, _, prListErr := c.Github.PullRequests.ListFiles(*c.Ctx, e.Repo.GetOwner().GetLogin(), e.Repo.GetName(), pr.GetNumber(), &github.ListOptions{})
		if prListErr != nil {
			return nil, errors.Wrap(prListErr, "Couldn't list files")
		}
		prFiles = append(prFiles, files...)
	}
	return prFiles, nil
}
