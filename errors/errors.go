package backoff

import (
	"net/http"

	"github.com/f2prateek/hickson"
)

func New(codes ...int) hickson.RetryPolicyFactory {
	return &factory{codes}
}

type factory struct {
	codes []int
}

func (f *factory) NewRetryPolicy(req *http.Request) hickson.RetryPolicy {
	codes := make(map[int]struct{}, len(f.codes))
	for _, c := range f.codes {
		codes[c] = struct{}{}
	}
	return &retryPolicy{
		retryC:  make(chan struct{}, 1),
		cancelC: make(chan struct{}, 1),
		codes:   codes,
	}
}

type retryPolicy struct {
	retryC  chan struct{}
	cancelC chan struct{}
	codes   map[int]struct{}
}

func (p *retryPolicy) Retry(req *http.Response, err error) <-chan struct{} {
	if _, ok := p.codes[req.StatusCode]; ok {
		p.retryC <- struct{}{}
	} else {
		p.cancelC <- struct{}{}
	}
	return p.retryC
}

func (p *retryPolicy) Cancel() <-chan struct{} {
	return p.cancelC
}

func (p *retryPolicy) Close() {
}
