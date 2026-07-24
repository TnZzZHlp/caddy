package dynamicproxy

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestUnmarshalCaddyfile(t *testing.T) {
	p := new(DynamicProxy)
	err := p.UnmarshalCaddyfile(caddyfile.NewTestDispenser(`dynamic_proxy {
		prefix /proxy/
	}`))
	if err != nil {
		t.Fatalf("UnmarshalCaddyfile() error = %v", err)
	}
	if p.Prefix != "/proxy/" {
		t.Fatalf("Prefix = %q, want %q", p.Prefix, "/proxy/")
	}
}

func TestUnmarshalCaddyfileRejectsInvalidPrefix(t *testing.T) {
	for name, input := range map[string]string{
		"missing":       `dynamic_proxy {}`,
		"relative":      `dynamic_proxy { prefix proxy/ }`,
		"not directory": `dynamic_proxy { prefix /proxy }`,
		"duplicate":     `dynamic_proxy { prefix /one/ prefix /two/ }`,
	} {
		t.Run(name, func(t *testing.T) {
			p := new(DynamicProxy)
			if err := p.UnmarshalCaddyfile(caddyfile.NewTestDispenser(input)); err == nil {
				t.Fatal("UnmarshalCaddyfile() error = nil, want error")
			}
		})
	}
}

func TestServeHTTPProxiesPathURL(t *testing.T) {
	type requestInfo struct {
		method string
		path   string
		host   string
		header string
		body   string
	}
	seen := make(chan requestInfo, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		seen <- requestInfo{
			method: r.Method,
			path:   r.URL.RequestURI(),
			host:   r.Host,
			header: r.Header.Get("X-Test-Header"),
			body:   string(body),
		}
		w.Header().Set("X-Upstream", "yes")
		_, _ = w.Write([]byte("proxied"))
	}))
	defer upstream.Close()

	handler := DynamicProxy{Prefix: "/proxy/"}
	target := upstream.URL + "/v1/resource?from=target"
	req := httptest.NewRequest(http.MethodPost, "http://proxy.test/proxy/"+target, strings.NewReader("payload"))
	req.Header.Set("X-Test-Header", "kept")
	req.RemoteAddr = "203.0.113.8:12345"
	rec := httptest.NewRecorder()

	if err := handler.ServeHTTP(rec, req, nil); err != nil {
		t.Fatalf("ServeHTTP() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "proxied" {
		t.Fatalf("body = %q, want %q", got, "proxied")
	}
	if got := rec.Header().Get("X-Upstream"); got != "yes" {
		t.Fatalf("X-Upstream = %q, want %q", got, "yes")
	}

	got := <-seen
	if got.method != http.MethodPost {
		t.Fatalf("upstream method = %q, want %q", got.method, http.MethodPost)
	}
	if got.path != "/v1/resource?from=target" {
		t.Fatalf("upstream URI = %q, want %q", got.path, "/v1/resource?from=target")
	}
	if got.host != strings.TrimPrefix(upstream.URL, "http://") {
		t.Fatalf("upstream Host = %q, want target host", got.host)
	}
	if got.header != "kept" {
		t.Fatalf("upstream X-Test-Header = %q, want %q", got.header, "kept")
	}
	if got.body != "payload" {
		t.Fatalf("upstream body = %q, want %q", got.body, "payload")
	}
}

func TestServeHTTPProxiesEscapedURL(t *testing.T) {
	seen := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.RequestURI()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	handler := DynamicProxy{Prefix: "/proxy/"}
	target := upstream.URL + "/encoded?source=path"
	req := httptest.NewRequest(http.MethodGet, "http://proxy.test/proxy/"+url.PathEscape(target), nil)
	rec := httptest.NewRecorder()

	if err := handler.ServeHTTP(rec, req, nil); err != nil {
		t.Fatalf("ServeHTTP() error = %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := <-seen; got != "/encoded?source=path" {
		t.Fatalf("upstream URI = %q, want %q", got, "/encoded?source=path")
	}
}

func TestServeHTTPRejectsInvalidTarget(t *testing.T) {
	handler := DynamicProxy{Prefix: "/proxy/"}
	req := httptest.NewRequest(http.MethodGet, "http://proxy.test/proxy/ftp://example.com/file", nil)
	err := handler.ServeHTTP(httptest.NewRecorder(), req, nil)

	var handlerErr caddyhttp.HandlerError
	if !errors.As(err, &handlerErr) {
		t.Fatalf("error = %v, want caddyhttp.HandlerError", err)
	}
	if handlerErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", handlerErr.StatusCode, http.StatusBadRequest)
	}
}
