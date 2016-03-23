package hickson

import "net/http"

// RetryTemporaryErrors returns a retry policy factory that retries any temporary
// errors.
func RetryTemporaryErrors() RetryPolicyFactory {
	return RetryPolicyFactoryFunc(func(r *http.Request) RetryPolicy {
		return &temporaryRetryPolicy{}
	})
}

// Interface for temporary errors.
type temporary interface {
	Temporary() bool
}

type temporaryRetryPolicy struct{}

func (p *temporaryRetryPolicy) Retry(resp *http.Response, err error) bool {
	if terr, ok := err.(temporary); ok {
		if terr.Temporary() {
			return true
		}
	}

	return false
}
