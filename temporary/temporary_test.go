package temporary_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/f2prateek/hickson/temporary"
)

type TemporaryError struct {
	temporary bool
}

func (t *TemporaryError) Temporary() bool {
	return t.temporary
}

func (t *TemporaryError) Error() string {
	return fmt.Sprintf("%v", t)
}

func TestTemporary(t *testing.T) {
	cases := []struct {
		err   error
		retry bool
	}{
		{&TemporaryError{true}, true},
		{&TemporaryError{false}, false},
		{errors.New("test"), false},
	}

	policy := temporary.TemporaryErrors().New(nil)

	for _, c := range cases {
		retry, err := policy.Retry(nil, c.err)
		assert.Equal(t, nil, err)
		assert.Equal(t, c.retry, retry)
	}
}
