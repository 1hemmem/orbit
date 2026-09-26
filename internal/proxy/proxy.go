package proxy

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"orbit/internal/balancer"
)

func NewHandler(rr *balancer.RoundRobin) http.Handler {
	transport := &http.Transport{
		DialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addr := rr.Next()
		if addr == "" {
			http.Error(w, "no healthy backends", http.StatusServiceUnavailable)
			return
		}
		rp := &httputil.ReverseProxy{
			Transport: transport,
			Director: func(req *http.Request) {
				target, _ := url.Parse("http://" + addr)
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
			},
			ModifyResponse: func(resp *http.Response) error {
				resp.Header.Set("X-Backend", resp.Request.URL.Host)
				return nil
			},
		}
		rp.ServeHTTP(w, r)
	})
}
