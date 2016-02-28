package temporary_test

import (
	"errors"
	"net"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/f2prateek/hickson/temporary"
)

func TestTemporary(t *testing.T) {
	cases := []struct {
		err   error
		retry bool
	}{
		{&net.DNSError{IsTemporary: true}, true},
		{&net.DNSError{IsTemporary: false}, false},
		{errors.New("test"), false},
	}

	policy := temporary.RetryErrors().New(nil)

	for _, c := range cases {
		retry := policy.Retry(nil, c.err)
		assert.Equal(t, c.retry, retry)
	}
}
