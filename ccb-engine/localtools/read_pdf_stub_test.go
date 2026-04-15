package localtools

import (
	"errors"
	"fmt"
	"testing"
)

func TestValidateReadPagesParameter(t *testing.T) {
	if err := ValidateReadPagesParameter(""); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReadPagesParameter("3"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReadPagesParameter("1-5"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReadPagesParameter("not-a-range"); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateReadPagesParameter("1-25"); err == nil {
		t.Fatal("expected range error")
	}
}

func TestPDFStubErrors(t *testing.T) {
	err := fmt.Errorf("%w", ErrReadPDFPagesNotImplementedInGo)
	if !errors.Is(err, ErrReadPDFPagesNotImplementedInGo) {
		t.Fatal(err)
	}
}
