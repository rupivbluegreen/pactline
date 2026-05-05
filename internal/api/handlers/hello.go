package handlers

import "net/http"

type HelloResponse struct {
	Message string `json:"message"`
}

func Hello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "world"
	}
	writeJSON(w, http.StatusOK, HelloResponse{Message: "hello, " + name})
}
