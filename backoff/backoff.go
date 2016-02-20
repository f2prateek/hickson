package backoff

import (
	"net/http"

	"github.com/f2prateek/hickson"
	"github.com/segmentio/backo-go"
)

var Default = New(backo.DefaultBacko())

func New(b *backo.Backo) hickson.RetryPolicyFactory {
	return &factory{b}
}

type factory struct {
	b *backo.Backo
}

func (f *factory) NewRetryPolicy(req *http.Request) hickson.RetryPolicy {
	ticker := f.b.NewTicker()
	return &retryPolicy{ticker, make(chan struct{})}
}

type retryPolicy struct {
	ticker *backo.Ticker
	done   chan struct{}
}

func (p *retryPolicy) Retry(req *http.Response, err error) <-chan struct{} {
	out := make(chan struct{})
	go func() {
		for {
			select {
			case <-p.ticker.C:
				out <- struct{}{}
			case <-p.done:
				p.ticker.Stop()
				return
			}
		}
	}()
	return out
}

func (p *retryPolicy) Cancel() <-chan struct{} {
	return make(chan struct{})
}

func (p *retryPolicy) Close() {
	p.done <- struct{}{}
}
