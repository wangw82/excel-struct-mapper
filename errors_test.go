package mapper

import (
	"errors"
	"testing"
)

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("cause")
	err := locatedError(KindConversion, "Data", "items", "ID", 1, 27, cause)
	if !errors.Is(err, cause) {
		t.Fatal("cause not unwrapped")
	}
	var mapped *Error
	if !errors.As(err, &mapped) || mapped.Row != 2 || mapped.Col != 28 || mapped.Cell != "AB2" {
		t.Fatalf("error = %#v", err)
	}
}

func TestValidationIssue(t *testing.T) {
	cause := errors.New("failed")
	issue := ValidationIssue{Field: "ID", Rule: "positive", Cause: cause}
	if !errors.Is(issue, cause) || issue.Error() == "" {
		t.Fatalf("issue = %v", issue)
	}
	if (ValidationIssue{Field: "ID", Rule: "required"}).Error() == "" {
		t.Fatal("empty issue error")
	}
}
