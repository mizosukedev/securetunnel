package server

import (
	"net/http"
)

// Authorizer is an interface that allow to change the functions related to authentication.
type Authorizer interface {
	Pluggable
	// Login the service and return your credentials.
	Login(http.ResponseWriter, *http.Request)
	// Logout the server.
	Logout(http.ResponseWriter, *http.Request)
	// HasAuth Check if the request has the credentials and make sure that
	// the credentials are authorized to access the resource.
	HasAuth(*http.Request) error
}

// NilAuthorizer is an interface that implements Authorizer.
// NilAuthorizer does not authenticate. It is used only for development purposes.
type NilAuthorizer struct {
}

func (auth *NilAuthorizer) Init() error {
	return nil
}

func (auth *NilAuthorizer) Login(writer http.ResponseWriter, request *http.Request) {

	writer.WriteHeader(http.StatusOK)
}

func (auth *NilAuthorizer) Logout(writer http.ResponseWriter, request *http.Request) {

	writer.WriteHeader(http.StatusOK)
}

func (auth *NilAuthorizer) HasAuth(*http.Request) error {

	return nil
}
