package web

import (
	"github.com/valyala/fasthttp"
	"strings"
	"time"
)

type Web struct {
	client *fasthttp.Client
}

func New(name string) *Web {
	w := &Web{
		client: &fasthttp.Client{
			Name: name,
		},
	}
	return w
}

func (w *Web) Get(url string, headers, queries map[string]string, timeout time.Duration) (body []byte, code int, err error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(fasthttp.MethodGet)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if len(queries) > 0 {
		builder := strings.Builder{}
		builder.WriteString(url)
		builder.WriteString("?")
		for k, v := range queries {
			builder.WriteString(k)
			builder.WriteString("=")
			builder.WriteString(v)
			builder.WriteString("&")
		}
		url = builder.String()
	}
	req.Header.SetRequestURI(url)
	resp := fasthttp.AcquireResponse()
	if timeout > 0 {
		err = w.client.DoTimeout(req, resp, timeout)
		if err != nil {
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			return
		}
	} else {
		if err = w.client.Do(req, resp); err != nil {
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			return
		}
	}
	fasthttp.ReleaseRequest(req)
	code, body = resp.StatusCode(), make([]byte, len(resp.Body()))
	copy(body, resp.Body())
	fasthttp.ReleaseResponse(resp)
	return
}

func (w *Web) Post(url string, headers map[string]string, payload []byte, timeout time.Duration) (body []byte, code int, err error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetRequestURI(url)
	req.SetBody(payload)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp := fasthttp.AcquireResponse()
	if timeout > 0 {
		if err = w.client.DoTimeout(req, resp, timeout); err != nil {
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			return
		}
	} else {
		if err = w.client.Do(req, resp); err != nil {
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			return
		}
	}
	fasthttp.ReleaseRequest(req)
	code, body = resp.StatusCode(), make([]byte, len(resp.Body()))
	copy(body, resp.Body())
	fasthttp.ReleaseResponse(resp)
	return
}
