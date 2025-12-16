package api

import (
	"net/http"
)

type API interface {
	initRoutesV1()
	GetInternalHandler(string) (string, map[string]string, error)
}

type APIHandler struct {
	API API
}

type Route struct {
	Path            string
	Method          string
	HTTPHandler     http.HandlerFunc
	InternalHandler string
}
