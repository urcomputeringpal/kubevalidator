package validator

import (
	"fmt"

	"github.com/google/go-github/github"
)

// Annotations is an array of pointers to CheckRunAnnotations
type Annotations []*github.CheckRunAnnotation

func (a Annotations) Len() int {
	return len(a)
}
func (a Annotations) Swap(i, j int) {
	*a[i], *a[j] = *a[j], *a[i]
}
func (a Annotations) Less(i, j int) bool {
	one := fmt.Sprintf("%d:%s", a[i].GetStartLine(), a[i].GetMessage())
	two := fmt.Sprintf("%d:%s", a[j].GetStartLine(), a[j].GetMessage())
	return one < two
}
