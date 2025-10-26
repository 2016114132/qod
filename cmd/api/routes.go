package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	// setup a new router
	router := httprouter.New()
	// handle 404
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	// handle 405
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)
	// setup routes
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodPost,
		"/v1/quotes",
		app.requireActivatedUser(app.requirePermission("quotes:write", app.createQuoteHandler)))

	router.HandlerFunc(http.MethodGet,
		"/v1/quotes/:id",
		app.requireActivatedUser(app.requirePermission("quotes:read", app.displayQuoteHandler)))

	router.HandlerFunc(http.MethodPatch,
		"/v1/quotes/:id",
		app.requireActivatedUser(app.requirePermission("quotes:write", app.updateQuoteHandler)))

	router.HandlerFunc(http.MethodDelete,
		"/v1/quotes/:id",
		app.requireActivatedUser(app.requirePermission("quotes:write", app.deleteQuoteHandler)))

	router.HandlerFunc(http.MethodGet,
		"/v1/quotes",
		app.requireActivatedUser(app.requirePermission("quotes:read", app.listQuotesHandler)))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)

	// We use PUT instead of POST because PUT is idempotent
	// and appropriate for this endpoint.  The activation
	// should only happens once, also we are not creating a resource
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	// Request sent first to recoverPanic()
	// then sent to enableCORS()
	// then sent to rateLimit()
	// finally it is sent to the router.
	return app.recoverPanic(app.enableCORS(app.rateLimit((app.authenticate(router)))))
}
