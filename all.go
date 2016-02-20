package hickson

import (
	"net/http"
	"sync"
)

func RetryAll(factories ...RetryPolicyFactory) RetryPolicyFactory {
	return &retryAllPolicyFactory{factories}
}

type retryAllPolicyFactory struct {
	factories []RetryPolicyFactory
}

func (f *retryAllPolicyFactory) NewRetryPolicy(req *http.Request) RetryPolicy {
	policies := make([]RetryPolicy, len(f.factories))
	for _, policyFactory := range f.factories {
		policies = append(policies, policyFactory.NewRetryPolicy(req))
	}
	return &retryAllPolicy{policies}
}

type retryAllPolicy struct {
	policies []RetryPolicy
}

func (all *retryAllPolicy) Retry(req *http.Response, err error) <-chan struct{} {
	// Retry emits on out once *all* of the policies emit on their retry channel.
	out := make(chan struct{})

	go func() {
		for {
			var wg sync.WaitGroup
			for _, policy := range all.policies {
				wg.Add(1)
				go func(policy RetryPolicy) {
					defer wg.Done()
					<-policy.Retry(req, err)
				}(policy)
			}
			wg.Wait()
			out <- struct{}{}
		}
	}()

	return out
}

func (p *retryAllPolicy) Cancel() <-chan struct{} {
	// Cancel emits on out once *any* of the policies emit on their cancel channel.
	out := make(chan struct{})

	go func() {
		for {
			merged := make(chan struct{}, len(p.policies))
			done := make(chan struct{})
			defer close(done)

			for _, policy := range p.policies {
				go func(policy RetryPolicy) {
					select {
					case <-policy.Cancel():
						merged <- struct{}{}
					case <-done:
						return
					}
				}(policy)
			}

			out <- <-merged
		}
	}()

	return out
}

func (p *retryAllPolicy) Close() {
	for _, policy := range p.policies {
		policy.Close()
	}
}
