package hickson_test

import (
	"errors"
	"log"
	"net/http"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/f2prateek/hickson"
	"github.com/f2prateek/train"
	"github.com/stretchr/testify/mock"
)

// errInterceptor shorts the chain and returns a test error.
var errInterceptor = train.InterceptorFunc(func(chain train.Chain) (*http.Response, error) {
	return nil, errors.New("test: short error")
})

func TestHickson(t *testing.T) {
	mockPolicy := new(MockPolicy)
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(false, nil).Once()
	mockPolicyFactory := mockFactory(mockPolicy)
	client := &http.Client{
		Transport: train.Transport(hickson.New(mockPolicyFactory), errInterceptor),
	}

	resp, err := client.Get("https://golang.org/")

	mockPolicy.AssertExpectations(t)
	assert.Equal(t, true, resp == nil)
	assert.Equal(t, "Get https://golang.org/: retries exhausted", err.Error())
}

func TestErrorsArePropogated(t *testing.T) {
	mockPolicy := new(MockPolicy)
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(false, errors.New("test: policy error")).Once()
	mockPolicyFactory := mockFactory(mockPolicy)
	client := &http.Client{
		Transport: train.Transport(hickson.New(mockPolicyFactory), errInterceptor),
	}

	resp, err := client.Get("https://golang.org/")

	mockPolicy.AssertExpectations(t)
	assert.Equal(t, true, resp == nil)
	assert.Equal(t, "Get https://golang.org/: test: policy error", err.Error())
}

func TestMax(t *testing.T) {
	mockPolicy := new(MockPolicy)
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()
	mockPolicyFactory := mockFactory(mockPolicy)
	client := &http.Client{
		Transport: train.Transport(hickson.New(hickson.RetryMax(2, mockPolicyFactory)), errInterceptor),
	}

	resp, err := client.Get("https://golang.org/")

	mockPolicy.AssertExpectations(t)
	assert.Equal(t, true, resp == nil)
	assert.Equal(t, "Get https://golang.org/: retries exhausted", err.Error())
}

func TestAllRetries(t *testing.T) {
	policy1 := new(MockPolicy)
	policy1.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()
	policy1.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()

	policy2 := new(MockPolicy)
	policy2.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()
	policy2.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()

	factory := hickson.RetryMax(2, hickson.RetryAll(mockFactory(policy1), mockFactory(policy2)))
	client := &http.Client{
		Transport: train.Transport(hickson.New(factory), errInterceptor),
	}

	resp, err := client.Get("https://golang.org/")

	policy1.AssertExpectations(t)
	policy2.AssertExpectations(t)
	assert.Equal(t, true, resp == nil)
	assert.Equal(t, "Get https://golang.org/: retries exhausted", err.Error())
}

func TestAllRetriesFailsWithFirstPolicy(t *testing.T) {
	policy1 := new(MockPolicy)
	policy1.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()
	policy1.On("Retry", mock.Anything, mock.Anything).Return(true, errors.New("test: policy1 error")).Once()

	policy2 := new(MockPolicy)
	policy2.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()

	factory := hickson.RetryAll(mockFactory(policy1), mockFactory(policy2))
	client := &http.Client{
		Transport: train.Transport(hickson.New(factory), errInterceptor),
	}

	resp, err := client.Get("https://golang.org/")

	policy1.AssertExpectations(t)
	policy2.AssertExpectations(t)
	assert.Equal(t, true, resp == nil)
	assert.Equal(t, "Get https://golang.org/: test: policy1 error", err.Error())
}

func TestAllRetriesCanStop(t *testing.T) {
	policy1 := new(MockPolicy)
	policy1.On("Retry", mock.Anything, mock.Anything).Return(true, nil).Once()

	policy2 := new(MockPolicy)
	policy2.On("Retry", mock.Anything, mock.Anything).Return(false, nil).Once()

	factory := hickson.RetryAll(mockFactory(policy1), mockFactory(policy2))
	client := &http.Client{
		Transport: train.Transport(hickson.New(factory), errInterceptor),
	}

	resp, err := client.Get("https://golang.org/")

	policy1.AssertExpectations(t)
	policy2.AssertExpectations(t)
	assert.Equal(t, true, resp == nil)
	assert.Equal(t, "Get https://golang.org/: retries exhausted", err.Error())
}

// Mocks.

func mockFactory(p hickson.RetryPolicy) hickson.RetryPolicyFactory {
	return hickson.RetryPolicyFactoryFunc(func(r *http.Request) hickson.RetryPolicy {
		return p
	})
}

type MockPolicy struct {
	mock.Mock
}

func getBool(i interface{}) bool {
	if b, ok := i.(bool); ok {
		return b
	}
	return false
}

func getError(i interface{}) error {
	if e, ok := i.(error); ok {
		return e
	}
	return nil
}

func (p *MockPolicy) Retry(resp *http.Response, respErr error) (bool, error) {
	log.Println("Retry:", resp, respErr)
	args := p.Called(resp, respErr)
	return getBool(args.Get(0)), getError(args.Get(1))
}
