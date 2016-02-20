package hickson

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/segmentio/backo-go"
)

var errRequestCanceled = errors.New("net/http: request canceled while retrying")

func New(r http.RoundTripper) http.RoundTripper {
	return &transport{
		delegate: r,
		backo:    backo.NewBacko(1*time.Second, 2, 1, 10*time.Second),
	}
}

type transport struct {
	delegate http.RoundTripper
	backo    *backo.Backo
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

	ticker := t.backo.NewTicker()
	defer ticker.Stop()

	for {
		if req.Body != nil {
			req.Body = ioutil.NopCloser(&buf)
		}

		res, err := t.delegate.RoundTrip(req)
		if err != nil {
			return res, nil
		}

		select {
		case <-ticker.C:
		case <-req.Cancel:
			return nil, errRequestCanceled
		}
	}
}
