package main

import (
	"errors"
	"net/http"

	"github.com/2016114132/qod/internal/data"
	"github.com/2016114132/qod/internal/validator"
)

func (a *application) registerUserHandler(w http.ResponseWriter,
	r *http.Request) {
	// Get the passed in data from the request body and store in a temporary struct
	var incomingData struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := a.readJSON(w, r, &incomingData)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}
	// we will add the password later after we have hashed it
	user := &data.User{
		Username:  incomingData.Username,
		Email:     incomingData.Email,
		Activated: false,
	}
	// hash the password and store it along with the cleartext version
	err = user.Password.Set(incomingData.Password)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}
	// Perform validation for the User
	v := validator.New()

	data.ValidateUser(v, user)
	if !v.IsEmpty() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = a.userModel.Insert(user) // we will add userModel to main() later
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			a.failedValidationResponse(w, r, v.Errors)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}
	response := envelope{
		"user": user,
	}
	// Status code 201 resource created
	err = a.writeJSON(w, http.StatusCreated, response, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}
}
