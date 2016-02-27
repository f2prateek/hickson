package hickson

import "net/http"

// All returns a RetryPolicyFactory whose policy is to retry only if all the
// policies agree to retry. If any of the policies do not want to retry or
// return an error, this policy will do the same. Policies are consulted in the
// order they are provided.
func RetryAll(factories ...RetryPolicyFactory) RetryPolicyFactory {
	return RetryPolicyFactoryFunc(func(r *http.Request) RetryPolicy {
		policies := make([]RetryPolicy, 0)
		for _, f := range factories {
			policy := f.New(r)
			policies = append(policies, policy)
		}

		return &allPolicy{
			policies: policies,
		}
	})
}

type allPolicy struct {
	policies []RetryPolicy
}

func (p *allPolicy) Retry(resp *http.Response, respErr error) (bool, error) {
	for _, policy := range p.policies {
		retry, err := policy.Retry(resp, respErr)
		if err != nil {
			return false, err
		}
		if !retry {
			return false, nil
		}
	}
	return true, nil
}
