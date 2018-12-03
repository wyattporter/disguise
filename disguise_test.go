package main

import "bytes"
import "context"
import "crypto/hmac"
import "crypto/sha1"
import "encoding/hex"
import "net/http"
import "net/http/httptest"
import "os"
import "strings"
import "testing"
import "time"

import "github.com/stretchr/testify/suite"

func TestServe(t *testing.T) {
	suite.Run(t, new(ServeTestSuite))
}

type ServeTestSuite struct {
	suite.Suite

	d disguise
}

func (t *ServeTestSuite) SetupTest() {
	network = "unix"
	address = "@disguise_TestServe"
	timeout = time.Millisecond
}

func (t *ServeTestSuite) Test_httpServer_nil() {
	s := make(chan os.Signal)
	defer close(s)
	t.d = disguise{}
	t.Require().NotNil(t.d.Serve(s))
}

func (t *ServeTestSuite) Test_abort_on_signal() {
	s := make(chan os.Signal)
	defer close(s)
	t.d = disguise{
		Server: &http.Server{
			Addr:    address,
			Handler: handler,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s <- os.Interrupt
		cancel()
	}()
	err := t.d.Serve(s)
	<-ctx.Done()
	t.Require().Nil(err)
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

type HandlerTestSuite struct {
	suite.Suite

	s *httptest.Server
	r *httptest.Server
}

func (t *HandlerTestSuite) SetupTest() {
	secret = []byte("secret")
	t.s = httptest.NewServer(handler)
	t.r = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimLeft(r.URL.Path, "/") {
		case "text":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		case "html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case "image":
			w.Header().Set("Content-Type", "image/png")
		default:
			http.Error(w, "", http.StatusNotFound)
		}
	}))
}

func (t *HandlerTestSuite) TearDownTest() {
	t.s.Close()
	t.r.Close()
}

func (t *HandlerTestSuite) Test_200_for_valid_requests() {
	for _, s := range []string{
		"/image",
	} {
		mac := hmac.New(sha1.New, secret)
		n, err := mac.Write([]byte(t.r.URL + s))
		t.Require().Equal(len([]byte(t.r.URL+s)), n)
		t.Require().Nil(err)

		digest := string(hex.EncodeToString(mac.Sum(nil)))

		res, err := http.Get(strings.Join([]string{t.s.URL, digest, string(hex.EncodeToString([]byte(t.r.URL + s)))}, "/"))
		t.Require().NotNil(res)
		t.Require().Nil(err)
		t.Require().Equal(http.StatusOK, res.StatusCode)
	}
}

func (t *HandlerTestSuite) Test_400_for_bad_api_requests() {
	for _, path := range []string{
		"/a/b", "a/b", "c/d",
	} {
		res, err := http.Get(strings.Join([]string{t.s.URL, path}, "/"))
		t.Require().NotNil(res)
		t.Require().Nil(err)
		t.Require().Equal(http.StatusBadRequest, res.StatusCode)
	}
}

func (t *HandlerTestSuite) Test_404_for_non_api_requests() {
	for _, path := range []string{
		"", "something", "something else",
	} {
		res, err := http.Get(strings.Join([]string{t.s.URL, path}, "/"))
		t.Require().NotNil(res)
		t.Require().Nil(err)
		t.Require().Equal(http.StatusNotFound, res.StatusCode)
	}
}

func (t *HandlerTestSuite) Test_401_for_digest_url_mismatch() {
	for _, path := range []string{
		"0a/b0", "0000/aaaa", "0123/4567",
	} {
		res, err := http.Get(strings.Join([]string{t.s.URL, path}, "/"))
		t.Require().NotNil(res)
		t.Require().Nil(err)
		t.Require().Equal(http.StatusUnauthorized, res.StatusCode)
	}
}

func (t *HandlerTestSuite) Test_405_for_bad_methods() {
	res, err := http.Post(t.s.URL, "", bytes.NewReader([]byte("")))
	t.Require().NotNil(res)
	t.Require().Nil(err)
	t.Require().Equal(http.StatusMethodNotAllowed, res.StatusCode)

	res, err = http.Head(t.s.URL)
	t.Require().NotNil(res)
	t.Require().Nil(err)
	t.Require().Equal(http.StatusMethodNotAllowed, res.StatusCode)
}

func (t *HandlerTestSuite) Test_406_for_wrong_content_type() {
	for _, s := range []string{
		"/text", "/html",
	} {
		mac := hmac.New(sha1.New, secret)
		n, err := mac.Write([]byte(t.r.URL + s))
		t.Require().Equal(len([]byte(t.r.URL+s)), n)
		t.Require().Nil(err)

		digest := string(hex.EncodeToString(mac.Sum(nil)))

		res, err := http.Get(strings.Join([]string{t.s.URL, digest, string(hex.EncodeToString([]byte(t.r.URL + s)))}, "/"))
		t.Require().NotNil(res)
		t.Require().Nil(err)
		t.Require().Equal(http.StatusNotAcceptable, res.StatusCode)
	}
}
