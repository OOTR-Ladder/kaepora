// package ootrapi is an API client for the ootrandomizer.com HTTP API v2
package ootrapi

import (
	"net/http"
	"time"
)

const baseURL = "https://ootrandomizer.com/api/v2"

type API struct {
	http http.Client
	key  string
}

func New(key string) *API {
	return &API{
		key: key,
		http: http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
