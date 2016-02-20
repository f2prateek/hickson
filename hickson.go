package hickson

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
)

type RetryPolicy interface {
	Retry(*http.Response, error) <-chan struct{}
	Cancel() <-chan struct{}
	Close()
}

type RetryPolicyFactory interface {
	NewRetryPolicy(*http.Request) RetryPolicy
}

var errRequestCanceled = errors.New("net/http: request canceled while retrying")
var errRetriesExhausted = errors.New("hickson: retries exhausted")

func New(delegate http.RoundTripper, retryPolicyFactory RetryPolicyFactory) http.RoundTripper {
	return &transport{
		delegate:           delegate,
		retryPolicyFactory: retryPolicyFactory,
	}
}

type transport struct {
	delegate           http.RoundTripper
	retryPolicyFactory RetryPolicyFactory
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var buf bytes.Buffer
	if req.Body != nil {
		if _, err := io.Copy(&buf, req.Body); err != nil {
			req.Body.Close()
			return nil, err
		}
		req.Body.Close()
	}

	retryPolicy := t.retryPolicyFactory.NewRetryPolicy(req)
	defer retryPolicy.Close()

	for {
		if req.Body != nil {
			req.Body = ioutil.NopCloser(&buf)
		}

		res, err := t.delegate.RoundTrip(req)

		select {
		case <-retryPolicy.Retry(res, err):
		case <-retryPolicy.Cancel():
			return res, err
		case <-req.Cancel:
			return nil, errRequestCanceled
		}
	}
}
