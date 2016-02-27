package hickson

import "net/http"

// Max wraps the given factory so that policies created will only be called at
// most `maxAttempts` times.
func RetryMax(maxAttempts int, factory RetryPolicyFactory) RetryPolicyFactory {
	return RetryPolicyFactoryFunc(func(r *http.Request) RetryPolicy {
		return &maxAttemptsPolicy{
			maxAttempts: maxAttempts,
			delegate:    factory.New(r),
		}
	})
}

type maxAttemptsPolicy struct {
	maxAttempts int
	attempts    int
	delegate    RetryPolicy
}

func (p *maxAttemptsPolicy) Retry(resp *http.Response, respErr error) (bool, error) {
	if p.attempts >= p.maxAttempts {
		return false, nil
	}

	p.attempts++
	return p.delegate.Retry(resp, respErr)
}
