package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/DextaAfrica/Backend/internal/apperror"
	"github.com/DextaAfrica/Backend/internal/transport/http/response"
)

// visitor tracks the token bucket for a single client IP plus when it was
// last seen, so idle entries can be evicted instead of growing the map
// forever under a botnet-style scan.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter throttles public, unauthenticated endpoints (enquiry,
// newsletter signup) per client IP to blunt basic abuse and scraping without
// requiring an external dependency like Redis for a single-instance
// deployment. It is intentionally in-memory and best-effort — if the API is
// ever run with more than one replica behind a load balancer, this should be
// swapped for a shared store, but the interface below keeps that swap local
// to this file.
type IPRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      rate.Limit
	burst    int
}

func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	l := &IPRateLimiter{
		visitors: make(map[string]*visitor),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
	go l.cleanupLoop()
	return l
}

func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, exists := l.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(l.rps, l.burst)
		l.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (l *IPRateLimiter) cleanupLoop() {
	for range time.Tick(time.Minute) {
		l.mu.Lock()
		for ip, v := range l.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !l.getLimiter(ip).Allow() {
			response.Error(w, r, apperror.RateLimited())
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if idx := len(fwd); idx > 0 {
			for i, c := range fwd {
				if c == ',' {
					return fwd[:i]
				}
			}
			return fwd
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
