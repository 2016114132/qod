package main

import (
	"fmt"
	"net/http"
)

func (a *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// defer will be called when the stack unwinds
		defer func() {
			// recover() checks for panics
			err := recover()
			if err != nil {
				w.Header().Set("Connection", "close")
				a.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// SCENARIO #1: Let's assume our API is a public API. Anyone can use it once they
// signup. So for the clients' browsers to be able to read our API responses, we
// need to set the Access-Control-Allow-Origin header to everyone
// (using the * operator). Notice we are back to returning 'http.Handler'
// Also notice that this middleware sets the header on the response object (w). We
// set the response header early in the middleware chain to enable our
// response to be accepted by the client's browser

func (a *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This header MUST be added to the response object or we defeat the whole
		// point of CORS. Why? Browsers want to be fast, so they cache stuff. If
		// on one response we say that  appletree.com is a trusted origin, the
		// browser is tempted to cache this, so if later a response comes
		// in from a different origin (evil.com), the browser will be tempted
		// to look in its cache and do what it did for the last response that
		// came in - allow it which would be bad and send the same response.
		// such as maybe display your account balance. We want to tell the browser
		// that the trusted origins might change so don't rely on the cache
		w.Header().Add("Vary", "Origin")
		// Let's check the request origin to see if it's in the trusted list
		origin := r.Header.Get("Origin")

		// Once we have a origin from the request header we need need to check
		if origin != "" {
			for i := range a.config.cors.trustedOrigins {
				if origin == a.config.cors.trustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
