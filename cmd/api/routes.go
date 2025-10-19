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
	router.HandlerFunc(http.MethodPost, "/v1/quotes", app.createQuoteHandler)
	router.HandlerFunc(http.MethodGet, "/v1/quotes/:id", app.displayQuoteHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/quotes/:id", app.updateQuoteHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/quotes/:id", app.deleteQuoteHandler)
	router.HandlerFunc(http.MethodGet, "/v1/quotes", app.listQuotesHandler)

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)

	// We use PUT instead of POST because PUT is idempotent
	// and appropriate for this endpoint.  The activation
	// should only happens once, also we are not creating a resource
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	// Request sent first to recoverPanic()
	// then sent to enableCORS()
	// then sent to rateLimit()
	// finally it is sent to the router.
	return app.recoverPanic(app.enableCORS(app.rateLimit(router)))
}
