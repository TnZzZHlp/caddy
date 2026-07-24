// Package dynamicproxy implements Caddy's dynamic_proxy HTTP handler.
//
// The handler treats the portion of a request path after Prefix as an
// absolute http(s) URL and proxies the request to that URL. For example:
//
//	dynamic_proxy {
//		prefix /proxy/
//	}
//
// turns /proxy/https://example.com/assets/app.js into a request to
// https://example.com/assets/app.js.
package dynamicproxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(DynamicProxy{})
	httpcaddyfile.RegisterHandlerDirective("dynamic_proxy", parseCaddyfile)
}

// DynamicProxy proxies a request to the absolute URL encoded in its path.
// Prefix is the path segment to remove before parsing that URL.
type DynamicProxy struct {
	Prefix string `json:"prefix,omitempty"`
}

// CaddyModule returns this module's Caddy registration information.
func (DynamicProxy) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.dynamic_proxy",
		New: func() caddy.Module { return new(DynamicProxy) },
	}
}

// Provision validates JSON configuration before the handler starts serving.
func (p *DynamicProxy) Provision(_ caddy.Context) error {
	return p.validate()
}

// ServeHTTP sends the request to the absolute HTTP(S) URL after Prefix.
// It is terminal by design: a successful proxy response is written directly
// to the client instead of invoking the next handler.
func (p *DynamicProxy) ServeHTTP(w http.ResponseWriter, r *http.Request, _ caddyhttp.Handler) error {
	target, err := p.targetURL(r)
	if err != nil {
		return caddyhttp.Error(http.StatusBadRequest, err)
	}

	// ReverseProxy handles hop-by-hop headers, streaming, upgrades, trailers,
	// and X-Forwarded-* headers correctly. Give it a copy so it cannot mutate
	// the parsed target while processing the request.
	targetCopy := *target
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL = &targetCopy
			pr.Out.Host = targetCopy.Host
			pr.Out.RequestURI = ""
			pr.SetXForwarded()
		},
		ErrorHandler: func(rw http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(rw, "dynamic proxy: upstream request failed", http.StatusBadGateway)
		},
	}

	proxy.ServeHTTP(w, r)
	return nil
}

// UnmarshalCaddyfile implements the Caddyfile syntax:
//
//	dynamic_proxy {
//		prefix <path-prefix>
//	}
func (p *DynamicProxy) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume "dynamic_proxy"
	if d.NextArg() {
		return d.ArgErr()
	}

	for d.NextBlock(0) {
		switch d.Val() {
		case "prefix":
			if p.Prefix != "" {
				return d.Err("prefix already specified")
			}
			if !d.AllArgs(&p.Prefix) {
				return d.ArgErr()
			}
		default:
			return d.Errf("unrecognized subdirective %q", d.Val())
		}
	}

	return p.validate()
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	proxy := new(DynamicProxy)
	return proxy, proxy.UnmarshalCaddyfile(h.Dispenser)
}

func (p *DynamicProxy) targetURL(r *http.Request) (*url.URL, error) {
	if !strings.HasPrefix(r.URL.Path, p.Prefix) {
		return nil, fmt.Errorf("request path does not start with configured prefix")
	}

	rawTarget := strings.TrimPrefix(r.URL.Path, p.Prefix)
	if rawTarget == "" {
		return nil, fmt.Errorf("missing target URL after prefix")
	}

	target, err := url.Parse(rawTarget)
	if err != nil {
		return nil, fmt.Errorf("parse target URL: %w", err)
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, fmt.Errorf("target URL must use http or https")
	}
	if target.Host == "" {
		return nil, fmt.Errorf("target URL must include a host")
	}
	if target.Fragment != "" {
		return nil, fmt.Errorf("target URL must not include a fragment")
	}

	// A literal '?' terminates the inbound request URI, so preserve that query
	// as part of the target URL. A percent-encoded '?' is already in rawTarget
	// and is parsed above as a target query.
	if r.URL.RawQuery != "" {
		if target.RawQuery == "" {
			target.RawQuery = r.URL.RawQuery
		} else {
			target.RawQuery += "&" + r.URL.RawQuery
		}
		target.ForceQuery = target.ForceQuery || r.URL.ForceQuery
	}

	return target, nil
}

func (p *DynamicProxy) validate() error {
	if p.Prefix == "" {
		return fmt.Errorf("dynamic_proxy requires a prefix")
	}
	if !strings.HasPrefix(p.Prefix, "/") {
		return fmt.Errorf("dynamic_proxy prefix must start with '/'")
	}
	if !strings.HasSuffix(p.Prefix, "/") {
		return fmt.Errorf("dynamic_proxy prefix must end with '/'")
	}
	if strings.ContainsAny(p.Prefix, "?#") {
		return fmt.Errorf("dynamic_proxy prefix must not contain '?' or '#'")
	}
	return nil
}

// Interface guards.
var (
	_ caddy.Provisioner           = (*DynamicProxy)(nil)
	_ caddyhttp.MiddlewareHandler = (*DynamicProxy)(nil)
	_ caddyfile.Unmarshaler       = (*DynamicProxy)(nil)
)
