package hickson_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/f2prateek/hickson"
	"github.com/f2prateek/train"
	"github.com/gohttp/response"
	"github.com/stretchr/testify/mock"
)

// errInterceptor shorts the chain and returns a test error.
var errInterceptor = train.InterceptorFunc(func(chain train.Chain) (*http.Response, error) {
	return nil, errors.New("test: short error")
})

func TestHickson(t *testing.T) {
	mockPolicy := new(MockPolicy)
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(true).Once()
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(true).Once()
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(false).Once()
	mockPolicyFactory := mockFactory(mockPolicy)
	client := &http.Client{
		Transport: train.Transport(hickson.New(mockPolicyFactory), errInterceptor),
	}

	resp, err := client.Get("https://golang.org/")

	mockPolicy.AssertExpectations(t)
	assert.Equal(t, true, resp == nil)
	assert.Equal(t, "Get https://golang.org/: test: short error", err.Error())
}

func TestMax(t *testing.T) {
	mockPolicy := new(MockPolicy)
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(true).Once()
	mockPolicy.On("Retry", mock.Anything, mock.Anything).Return(true).Once()
	mockPolicyFactory := mockFactory(mockPolicy)
	client := &http.Client{
		Transport: train.Transport(hickson.New(hickson.RetryMax(2, mockPolicyFactory)), errInterceptor),
	}

	resp, err := client.Get("https://golang.org/")

	mockPolicy.AssertExpectations(t)
	assert.Equal(t, true, resp == nil)
	assert.Equal(t, "Get https://golang.org/: test: short error", err.Error())
}

func TestAllRetries(t *testing.T) {
	policy1 := new(MockPolicy)
	policy1.On("Retry", mock.Anything, mock.Anything).Return(true).Once()
	policy1.On("Retry", mock.Anything, mock.Anything).Return(true).Once()

	policy2 := new(MockPolicy)
	policy2.On("Retry", mock.Anything, mock.Anything).Return(true).Once()
	policy2.On("Retry", mock.Anything, mock.Anything).Return(true).Once()

	factory := hickson.RetryMax(2, hickson.RetryAll(mockFactory(policy1), mockFactory(policy2)))
	client := &http.Client{
		Transport: train.Transport(hickson.New(factory), errInterceptor),
	}

	resp, err := client.Get("https://golang.org/")

	policy1.AssertExpectations(t)
	policy2.AssertExpectations(t)
	assert.Equal(t, true, resp == nil)
	assert.Equal(t, "Get https://golang.org/: test: short error", err.Error())
}

func TestAllRetriesFailsWithFirstPolicy(t *testing.T) {
	policy1 := new(MockPolicy)
	policy1.On("Retry", mock.Anything, mock.Anything).Return(true).Once()
	policy1.On("Retry", mock.Anything, mock.Anything).Return(false).Once()

	policy2 := new(MockPolicy)
	policy2.On("Retry", mock.Anything, mock.Anything).Return(true).Once()

	factory := hickson.RetryAll(mockFactory(policy1), mockFactory(policy2))
	client := &http.Client{
		Transport: train.Transport(hickson.New(factory), errInterceptor),
	}

	resp, err := client.Get("https://golang.org/")

	policy1.AssertExpectations(t)
	policy2.AssertExpectations(t)
	assert.Equal(t, true, resp == nil)
	assert.Equal(t, "Get https://golang.org/: test: short error", err.Error())
}

func TestTemporary(t *testing.T) {
	cases := []struct {
		err   error
		retry bool
	}{
		{&net.DNSError{IsTemporary: true}, true},
		{&net.DNSError{IsTemporary: false}, false},
		{errors.New("test"), false},
	}

	policy := hickson.RetryTemporaryErrors().New(nil)

	for _, c := range cases {
		retry := policy.Retry(nil, c.err)
		assert.Equal(t, c.retry, retry)
	}
}

func ExampleNew() {
	// Our test server.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		response.OK(w, "Hello World!")
	}))
	defer ts.Close()

	// An interceptor that randomly injects a temporary error into the chain.
	rand.Seed(2) // Try changing this number!
	errInterceptor := train.InterceptorFunc(func(chain train.Chain) (*http.Response, error) {
		i := rand.Intn(4)
		if i != 2 {
			return nil, &net.DNSError{IsTemporary: true}
		}
		return chain.Proceed(chain.Request())
	})

	// An interceptor that retries any temporary errors a maximum of 5 times.
	h := hickson.New(hickson.RetryMax(5, hickson.RetryTemporaryErrors()))
	client := &http.Client{
		Transport: train.Transport(h, errInterceptor),
	}

	// Make the request. The retry logic is transparent to your caller.
	resp, _ := client.Get(ts.URL)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
	// Output: Hello World!
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

func (p *MockPolicy) Retry(resp *http.Response, respErr error) bool {
	args := p.Called(resp, respErr)
	return getBool(args.Get(0))
}
