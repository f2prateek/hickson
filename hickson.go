package hickson

import (
	"errors"
	"net/http"

	"github.com/f2prateek/train"
)

var (
	ErrRequestCanceled = errors.New("request canceled")
)

type RetryPolicyFactory interface {
	// New returns a retry policy for the given request.
	New(*http.Request) RetryPolicy
}

// The RetryPolicyFactoryFunc type is an adapter to allow the use of ordinary
// functions as retry policy factories. If f is a function with the appropriate
// signature, RetryPolicyFactoryFunc(f) is a RetryPolicyFactory that calls f.
type RetryPolicyFactoryFunc func(*http.Request) RetryPolicy

// New calls f(r).
func (f RetryPolicyFactoryFunc) New(r *http.Request) RetryPolicy {
	return f(r)
}

type RetryPolicy interface {
	// Retry returns true if the if the request should be retried, return false
	// otherwise.
	Retry(*http.Response, error) bool
}

// New returns an interceptor that handles retries.
func New(factory RetryPolicyFactory) train.Interceptor {
	return &hickson{factory}
}

type hickson struct {
	factory RetryPolicyFactory
}

func (h *hickson) Intercept(c train.Chain) (*http.Response, error) {
	req := c.Request()
	policy := h.factory.New(req)

	for {
		select {
		case <-req.Cancel:
			return nil, ErrRequestCanceled
		default:
		}

		resp, respErr := c.Proceed(req)
		retry := policy.Retry(resp, respErr)
		if !retry {
			return resp, respErr
		}
	}
}
