package middleware

import "net/http"

// SecurityHeaders applies enterprise-grade defensive HTTP response headers
// conforming to OWASP ASVS 4.0 and financial-grade API benchmarks.
// These headers mitigate Clickjacking (X-Frame-Options), MIME-Confusion (X-Content-Type-Options),
// Cross-Site Scripting (CSP / X-XSS-Protection), SSL Strip (HSTS), and PII Information Leaks.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		// Protect against Clickjacking attacks
		h.Set("X-Frame-Options", "DENY")

		// Prevent MIME-sniffing vulnerabilities
		h.Set("X-Content-Type-Options", "nosniff")

		// Modern standard disables buggy older browser XSS filters
		h.Set("X-XSS-Protection", "0")

		// Restrict referrer data leakage across origins
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Strict API Content-Security-Policy: deny arbitrary execution and framing, permit favicon/self-hosted images
		h.Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")

		// HTTP Strict Transport Security: enforce HTTPS for 2 years with subdomains and preload
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")

		// Zero hardware sensor / capability leakage
		h.Set("Permissions-Policy", "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")

		// Mask server implementation identity from vulnerability scanners
		h.Set("X-Permitted-Cross-Domain-Policies", "none")

		// Prevent cache poisoning of sensitive JSON responses by default for authenticated routes
		if r.Header.Get("Authorization") != "" {
			h.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
			h.Set("Pragma", "no-cache")
		}

		next.ServeHTTP(w, r)
	})
}
