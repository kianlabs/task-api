package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// RateLimit membatasi request per alamat IP memakai token bucket sederhana di
// memori. Cocok untuk satu instance; untuk multi-instance gunakan penyimpanan
// bersama.
func RateLimit(requestsPerSecond float64, burst int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	buckets := make(map[string]*bucket)
	lastCleanup := time.Now()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)

			mu.Lock()
			// Bersihkan bucket idle agar map tidak tumbuh tanpa batas.
			if time.Since(lastCleanup) > time.Minute {
				for k, b := range buckets {
					if time.Since(b.lastSeen) > 10*time.Minute {
						delete(buckets, k)
					}
				}
				lastCleanup = time.Now()
			}

			b, ok := buckets[key]
			now := time.Now()
			if !ok {
				b = &bucket{tokens: float64(burst), lastSeen: now}
				buckets[key] = b
			}
			elapsed := now.Sub(b.lastSeen).Seconds()
			b.tokens = min(float64(burst), b.tokens+elapsed*requestsPerSecond)
			b.lastSeen = now

			allowed := b.tokens >= 1
			if allowed {
				b.tokens--
			}
			mu.Unlock()

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"too many requests"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
