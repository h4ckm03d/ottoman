package cache_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/bukalapak/ottoman/cache"
	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	data := []string{
		"foo/bar",
		"api:foo/bar",
		"bar:foo/bar",
	}

	for i := range data {
		key := cache.Normalize(data[i], "")
		assert.Equal(t, "foo/bar", key)

		key = cache.Normalize(data[i], "api")
		assert.Equal(t, "api:foo/bar", key)

		key = cache.Normalize(data[i], "foo")
		assert.Equal(t, "foo:foo/bar", key)
	}
}

type Sample struct {
	data map[string]string
}

func NewReader() cache.WriteReader {
	return &Sample{data: map[string]string{
		"foo":     `{"foo":"bar"}`,
		"fox":     `{"fox":"baz"}`,
		"api:foo": `{"foo":"bar"}`,
		"baz":     `x`,
	}}
}

func (m *Sample) Name() string {
	return "cache/reader"
}

func (m *Sample) Write(key string, value []byte, expiration time.Duration) error {
	m.data[key] = string(value)
	return nil
}

func (m *Sample) Read(key string) ([]byte, error) {
	if v, ok := m.data[key]; ok {
		return []byte(v), nil
	}

	return nil, errors.New("unknown cache")
}

func (m *Sample) ReadMulti(keys []string) (map[string][]byte, error) {
	z := make(map[string][]byte, len(keys))

	for _, key := range keys {
		v, _ := m.Read(key)
		z[key] = []byte(v)
	}

	return z, nil
}

type XSample struct{}

func (m *XSample) Write(key string, value []byte, expiration time.Duration) error {
	return errors.New("example error from Write")
}

func (m *XSample) Read(key string) ([]byte, error) {
	return nil, errors.New("example error from Read")
}

func (m *XSample) ReadMulti(keys []string) (map[string][]byte, error) {
	return nil, errors.New("example error from ReadMulti")
}

func (m *XSample) Name() string {
	return "cache/x-reader"
}

func NewRequest(s string) *http.Request {
	r, _ := http.NewRequest("GET", s, nil)
	return r
}

type Match struct{}

func NewResolver() cache.Resolver {
	return &Match{}
}

func (m *Match) Resolve(key string, r *http.Request) (*http.Request, error) {
	req, _ := m.ResolveRequest(r)

	switch key {
	case "zoo", "bad":
		req.URL.Path = "/" + key
	case "api:zoo":
		req.URL.Path = "/zoo"
	default:
		return nil, errors.New("unknown cache")
	}

	return req, nil
}

func (m *Match) ResolveRequest(r *http.Request) (*http.Request, error) {
	req := new(http.Request)
	url := new(url.URL)

	*req = *r
	*url = *r.URL

	req.URL = url

	return req, nil
}

type FailureTransport struct{}

func (t *FailureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, errors.New("Connection failure")
}

func NewRemoteServer() *httptest.Server {
	fn := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/zoo":
			io.WriteString(w, `{"zoo":"zac"}`)
		case "/zab":
			io.WriteString(w, `remote-x`)
		case "/bad":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	return httptest.NewServer(http.HandlerFunc(fn))
}
