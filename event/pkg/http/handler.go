package http

import (
	"context"
	"encoding/json"
	"errors"

	http1 "net/http"

	"github.com/dizaknz/gkgraph/event/pkg/endpoint"
	"github.com/go-kit/kit/transport/http"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func makeAddHandler(m *mux.Router, endpoints endpoint.Endpoints, options []http.ServerOption) {
	m.Methods("POST").
		Path("/add").
		Handler(
			handlers.CORS(
				handlers.AllowedMethods([]string{"POST"}),
				handlers.AllowedOrigins([]string{"*"}),
			)(http.NewServer(
				endpoints.AddEndpoint,
				decodeAddRequest,
				encodeAddResponse,
				options...,
			)),
		)
}

func decodeAddRequest(_ context.Context, r *http1.Request) (interface{}, error) {
	req := endpoint.AddRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func encodeAddResponse(ctx context.Context, w http1.ResponseWriter, response interface{}) (err error) {
	if f, ok := response.(endpoint.Failure); ok && f.Failed() != nil {
		ErrorEncoder(ctx, f.Failed(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(response)
	return
}

func ErrorEncoder(_ context.Context, err error, w http1.ResponseWriter) {
	w.WriteHeader(err2code(err))
	json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}

func ErrorDecoder(r *http1.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

func err2code(err error) int {
	return http1.StatusInternalServerError
}

type errorWrapper struct {
	Error string `json:"error"`
}
