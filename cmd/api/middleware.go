package main

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/2016114132/qod/internal/data"
	"github.com/2016114132/qod/internal/validator"

	"net"
	"sync" // need this for the Mutex that we will need
	"time"

	"golang.org/x/time/rate"
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

		// The request method can vary so don't rely on cache
		w.Header().Add("Vary", "Access-Control-Request-Method")

		// Let's check the request origin to see if it's in the trusted list
		origin := r.Header.Get("Origin")

		// Once we have a origin from the request header we need need to check
		if origin != "" {
			for i := range a.config.cors.trustedOrigins {
				if origin == a.config.cors.trustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					// check if it is a Preflight CORS request
					if r.Method == http.MethodOptions &&
						r.Header.Get("Access-Control-Request-Method") != "" {
						w.Header().Set("Access-Control-Allow-Methods",
							"OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers",
							"Authorization, Content-Type")

						// we need to send a 200 OK status. Also since there
						// is no need to continue the middleware chain we
						// we leave  - remember it is not a real 'comments' request but
						// only a preflight CORS request
						w.WriteHeader(http.StatusOK)
						return
					}
					break
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (a *application) rateLimit(next http.Handler) http.Handler {
	// Define a rate limiter struct
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time // remove map entries that are stale
	}
	var mu sync.Mutex                      // use to synchronize the map
	var clients = make(map[string]*client) // the actual map
	// A goroutine to remove stale entries from the map
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock() // begin cleanup
			// delete any entry not seen in three minutes
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock() // finish clean up
		}
	}()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.config.limiter.enabled {
			// get the IP address
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				a.serverErrorResponse(w, r, err)
				return
			}

			mu.Lock() // exclusive access to the map
			// check if ip address already in map, if not add it
			_, found := clients[ip]
			if !found {
				clients[ip] = &client{limiter: rate.NewLimiter(
					rate.Limit(a.config.limiter.rps),
					a.config.limiter.burst)}
			}
			// Update the last seem for the client
			clients[ip].lastSeen = time.Now()

			// Check the rate limit status
			if !clients[ip].limiter.Allow() {
				mu.Unlock() // no longer need exclusive access to the map
				a.rateLimitExceededResponse(w, r)
				return
			}

			mu.Unlock() // others are free to get exclusive access to the map
		}
		next.ServeHTTP(w, r)
	})

}

func (a *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// This header tells the servers not to cache the response when
		// the Authorization header changes. This also means that the server is not
		// supposed to serve the same cached data to all users regardless of their
		// Authorization values. Each unique user gets their own cache entry
		w.Header().Add("Vary", "Authorization")

		// Get the Authorization header from the request. It should have the
		// Bearer token
		authorizationHeader := r.Header.Get("Authorization")

		// If there is no Authorization header then we have an Anonymous user
		if authorizationHeader == "" {
			r = a.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}
		// Bearer token present so parse it. The Bearer token is in the form
		// Authorization: Bearer IEYZQUBEMPPAKPOAWTPV6YJ6RM
		// We will implement invalidAuthenticationTokenResponse() later
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			a.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Get the actual token
		token := headerParts[1]
		// Validate
		v := validator.New()

		data.ValidateTokenPlaintext(v, token)
		if !v.IsEmpty() {
			a.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Get the user info associated with this authentication token
		user, err := a.userModel.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				a.invalidAuthenticationTokenResponse(w, r)
			default:
				a.serverErrorResponse(w, r, err)
			}
			return
		}
		// Add the retrieved user info to the context
		r = a.contextSetUser(r, user)

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

// This middleware checks if the user is authenticated (not anonymous)
func (a *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		user := a.contextGetUser(r)

		if user.IsAnonymous() {
			a.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// This middleware checks if the user is activated
// It call the authentication middleware to help it do its job
func (a *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		user := a.contextGetUser(r)

		if !user.Activated {
			a.inactiveAccountResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	//We pass the activation check middleware to the authentication
	// middleware to call (next) if the authentication check succeeds
	// In other words, only check if the user is activated if they are
	// actually authenticated.
	return a.requireAuthenticatedUser(fn)
}

// This middleware checks if the user has the right permissions
// We send the permission that is expected as an argument
func (a *application) requirePermission(permissionCode string, next http.HandlerFunc) http.HandlerFunc {

	fn := func(w http.ResponseWriter, r *http.Request) {
		user := a.contextGetUser(r)
		// get all the permissions associated with the user
		permissions, err := a.permissionModel.GetAllForUser(user.ID)
		if err != nil {
			a.serverErrorResponse(w, r, err)
			return
		}
		if !permissions.Include(permissionCode) {
			a.notPermittedResponse(w, r)
			return
		}
		// they are good. Let's keep going
		next.ServeHTTP(w, r)
	}

	return a.requireActivatedUser(fn)

}

// We will create a new type right before we use it in our metrics middleware
type metricsResponseWriter struct {
	wrapped       http.ResponseWriter // the original http.ResponseWriter
	statusCode    int                 // this will contain the status code we need
	headerWritten bool                // has the response headers already been written?
}

// Create an new instance of our custom http.ResponseWriter once
// we are provided with the original http.ResponseWriter. We will set
// the status code to 200 by default since that is what Golang does as well
// the headerWritten is false by default so no need to specify
func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		wrapped:    w,
		statusCode: http.StatusOK,
	}
}

// Remember that the http.Header type is a map (key: value) of the headers
// Our custom http.ResponseWriter does not need to change the way the Header()
// method works, so all we do is call the original http.ResponseWriter's Header()
// method when our custom http.ResponseWriter's Header() method is called
func (mw *metricsResponseWriter) Header() http.Header {
	return mw.wrapped.Header()
}

// Let's write the status code that is provided
// Again the original http.ResponseWriter's WriteHeader() methods knows
// how to do this
func (mw *metricsResponseWriter) WriteHeader(statusCode int) {
	mw.wrapped.WriteHeader(statusCode)
	// After the call to WriteHeader() returns, we record
	// the first status code for use in our metrics
	// NOTE: Because we only want the first status code sent, we will
	// ignore any other status code that gets written. For example,
	// mw.WriteHeader(404) followed by mw.WriteHeader(500). The client
	// will receive a 404, the 500 will never be sent
	if !mw.headerWritten {
		mw.statusCode = statusCode
		mw.headerWritten = true
	}
}

// The write() method simply calls the original http.ResponseWriter's
// Write() method which write the data to the connection
func (mw *metricsResponseWriter) Write(b []byte) (int, error) {
	mw.headerWritten = true
	return mw.wrapped.Write(b)
}

// We need a function to get the original http.ResponseWriter
func (mw *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return mw.wrapped
}

// this middleware will run for every request received
func (a *application) metrics(next http.Handler) http.Handler {
	// Setup our variable to track the metrics
	var (
		totalRequestsReceived           = expvar.NewInt("total_requests_received")
		totalResponsesSent              = expvar.NewInt("total_responses_sent")
		totalProcessingTimeMicroseconds = expvar.NewInt("total_processing_time_μs")
		totalResponsesSentByStatus      = expvar.NewMap("total_responses_sent_by_status")
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// start is when we receive the request and start processing it
		start := time.Now()
		// update our request received counter
		totalRequestsReceived.Add(1)

		// create a custom responseWriter
		mw := newMetricsResponseWriter(w)

		// we send our custom responseWriter down the middleware chain
		next.ServeHTTP(mw, r)

		// remember the middleware chain goes in both directions, so we can
		// do things when we return back to our middleware.We will increment
		// the responses sent counter
		totalResponsesSent.Add(1)
		// extract the status code for use in our metrics since we have returned
		// from the middleware chain. The map uses strings so we need to convert the
		// status codes from their integer values to strings
		totalResponsesSentByStatus.Add(strconv.Itoa(mw.statusCode), 1)

		// calculate the processing time for this request. Remember we set start
		// at the beginning, so now since we are back in the middleware we can
		// compute the time taken
		// calculate the processing time for this request. Remember we set start
		// at the beginning, so now since we are back in the middleware we can
		// compute the time taken
		duration := time.Since(start).Microseconds()
		totalProcessingTimeMicroseconds.Add(duration)
	})
}
