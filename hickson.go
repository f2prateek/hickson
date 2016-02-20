package hickson

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
)

type RetryPolicy interface {
	Retry() <-chan struct{}
	Cancel() <-chan struct{}
	Close()
}

type RetryPolicyFactory interface {
	NewRetryPolicy() RetryPolicy
}

var errRequestCanceled = errors.New("net/http: request canceled while retrying")
var errRetriesExhausted = errors.New("hickson: retries exhausted")

func New(r http.RoundTripper, retryPolicyFactory RetryPolicyFactory) http.RoundTripper {
	return &transport{
		delegate:           r,
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

	retryPolicy := t.retryPolicyFactory.NewRetryPolicy()
	defer retryPolicy.Close()

	for {
		if req.Body != nil {
			req.Body = ioutil.NopCloser(&buf)
		}

		res, err := t.delegate.RoundTrip(req)
		if err != nil {
			return res, nil
		}

		select {
		case <-retryPolicy.Retry():
		case <-retryPolicy.Cancel():
			return nil, errRetriesExhausted
		case <-req.Cancel:
			return nil, errRequestCanceled
		}
	}
}
