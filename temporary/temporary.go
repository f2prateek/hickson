package temporary

import (
	"net/http"

	"github.com/f2prateek/hickson"
)

// TemporaryErrorsRetryPolicy returns a retry policy that retries any temporary
// errors.
func TemporaryErrors() hickson.RetryPolicyFactory {
	return hickson.RetryPolicyFactoryFunc(func(r *http.Request) hickson.RetryPolicy {
		return &temporaryRetryPolicy{}
	})
}

// Interface for temporary errors.
type temporary interface {
	Temporary() bool
}

type temporaryRetryPolicy struct{}

func (p *temporaryRetryPolicy) Retry(resp *http.Response, err error) (bool, error) {
	if terr, ok := err.(temporary); ok {
		if terr.Temporary() {
			return true, nil
		}
	}

	return false, nil
}
