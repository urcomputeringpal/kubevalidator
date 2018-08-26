package validator

import "github.com/google/go-github/github"

// Candidates is an array of Candidate
type Candidates []Candidate

// LoadBytes loads all of the files from GitHub
func (c Candidates) LoadBytes() []*github.CheckRunAnnotation {
	var a []*github.CheckRunAnnotation
	for _, candidate := range c {
		annotation := candidate.LoadBytes()
		if annotation != nil {
			a = append(a, annotation)
		}
	}
	return a
}

// Validate runs kubeval on all candidates
func (c Candidates) Validate() []*github.CheckRunAnnotation {
	var a []*github.CheckRunAnnotation
	for _, candidate := range c {
		annotations := candidate.Validate()
		if annotations != nil {
			a = append(a, annotations...)
		}
	}
	return a
}
