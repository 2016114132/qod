package main

import (
	// "fmt"
	"encoding/json"
	"net/http"
)

func (app *application) healthcheckHandler(w http.ResponseWriter,
	r *http.Request) {

	data := map[string]string{
		"status":      "available",
		"environment": app.config.env,
		"version":     app.config.vrs,
	}

	jsResponse, err := json.Marshal(data)

	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
		return
	}

	jsResponse = append(jsResponse, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsResponse)
}
