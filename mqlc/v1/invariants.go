package v1

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"go.mondoo.io/mondoo/llx"
)

// An Invariant is a condition that we expect compiled code to hold.
// This is used to find inconsistencies in our compiler and not for
// meant to be user facing
type Invariant struct {
	ShortName   string
	Description string
	Issues      []string
	// Checker returns true if the invariant holds
	Checker func(*llx.CodeBundle) bool
}

type InvariantFailed struct {
	ShortName string
	Source    string
}

func (e InvariantFailed) Error() string {
	return fmt.Sprintf("Invariant %q failed: Source => \n%s", e.ShortName, e.Source)
}

type InvariantList []Invariant

func (l InvariantList) Check(cb *llx.CodeBundle) error {
	var err error
	for _, i := range l {
		if !i.Checker(cb) {
			err = multierror.Append(err, InvariantFailed{
				ShortName: i.ShortName,
				Source:    cb.Source,
			})
		}
	}

	return err
}

var Invariants = InvariantList{
	{
		ShortName: "return-entrypoints-singular",
		Description: `
			The return statement indicates that the following expression
			is to be used for the value of the block. Our execution code
			assumes that only 1 value will be reported for the block.

			This means that there can only be 1 entrypoint. Further, it
			also means that num_entrypoints + num_datapoints == 1. The
			restriction on datapoints is just an artifact of the way things
			are written and can be changed, however the entrypoint should
			be the return value. I mention this as a reminder that not all
			parts of this invariant need to be this way forever and can be
			changed
		`,
		Issues: []string{
			"https://gitlab.com/mondoolabs/mondoo/-/issues/716",
		},
		Checker: func(cb *llx.CodeBundle) bool {
			return checkReturnEntrypoints(cb.DeprecatedV5Code)
		},
	},
}

func checkReturnEntrypoints(code *llx.CodeV1) bool {
	if code.SingleValue {
		if len(code.Entrypoints)+len(code.Datapoints) != 1 {
			return false
		}
	}

	for _, c := range code.Functions {
		if checkReturnEntrypoints(c) == false {
			return false
		}
	}

	return true
}
