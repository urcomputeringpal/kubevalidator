package validator

import "github.com/google/go-github/github"

// Annotations is an array of pointers to CheckRunAnnotations
type Annotations []*github.CheckRunAnnotation

func (a Annotations) Len() int {
	return len(a)
}
func (a Annotations) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a Annotations) Less(i, j int) bool {
	return a[i].GetStartLine() < a[j].GetStartLine()
}
