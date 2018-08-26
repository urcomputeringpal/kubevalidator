package validator

import "sort"

// Candidates is an array of pointers to Candidates
type Candidates []*Candidate

// LoadBytes loads all of the files from GitHub
func (c *Candidates) LoadBytes() Annotations {
	var a Annotations
	for _, candidate := range *c {
		annotation := candidate.LoadBytes()
		if annotation != nil {
			a = append(a, annotation)
		}
	}
	sort.Sort(a)
	return a
}

// Validate runs kubeval on all candidates
func (c *Candidates) Validate() Annotations {
	var a Annotations
	for _, candidate := range *c {
		annotations := candidate.Validate()
		if annotations != nil {
			a = append(a, annotations...)
		}
	}
	sort.Sort(a)
	return a
}
