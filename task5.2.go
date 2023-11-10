package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// Bank represents your data model
type Bank struct {
	ID       int64     `json:"id"`
	Code     string    `json:"code"`
	Name     string    `json:"name"`
	Currency string    `json:"currency"`
	URL      string    `json:"url"`
	CreateAt time.Time `json:"create_at"`
	CreateBy string    `json:"create_by"`
	UpdateAt time.Time `json:"update_at"`
	UpdateBy string    `json:"update_by"`
}

// store represents your database access layer
type store struct {
	db *sql.DB
}

func (s *store) getNextPageBankByID(ctx context.Context, currentUpdateAt time.Time) (*Bank, error) {
	var nextBank Bank
	rawSQL := `
		SELECT 
			id,
			code,
			name,
			currency,
			url,
			create_at,
			create_by,
			update_at,
			update_by
		FROM bank
		WHERE update_at > ?
		ORDER BY update_at ASC
		LIMIT 1
	`

	err := s.db.QueryRowContext(ctx, rawSQL, currentUpdateAt).Scan(
		&nextBank.ID, &nextBank.Code, &nextBank.Name, &nextBank.Currency, &nextBank.URL,
		&nextBank.CreateAt, &nextBank.CreateBy, &nextBank.UpdateAt, &nextBank.UpdateBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no next page found")
		}
		return nil, errors.Wrap(err, "failed to get next page bank")
	}

	return &nextBank, nil
}

func (s *store) getPreviousPageBankByID(ctx context.Context, currentUpdateAt time.Time) (*Bank, error) {
	var prevBank Bank
	rawSQL := `
		SELECT 
			id,
			code,
			name,
			currency,
			url,
			create_at,
			create_by,
			update_at,
			update_by
		FROM bank
		WHERE update_at < ?
		ORDER BY update_at DESC
		LIMIT 1
	`

	err := s.db.QueryRowContext(ctx, rawSQL, currentUpdateAt).Scan(
		&prevBank.ID, &prevBank.Code, &prevBank.Name, &prevBank.Currency, &prevBank.URL,
		&prevBank.CreateAt, &prevBank.CreateBy, &prevBank.UpdateAt, &prevBank.UpdateBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no previous page found")
		}
		return nil, errors.Wrap(err, "failed to get previous page bank")
	}

	return &prevBank, nil
}

type service struct {
	store *store
}

func (s *service) NextPage(ctx context.Context, currentUpdateAt time.Time) (*Bank, error) {
	nextBank, err := s.store.getNextPageBankByID(ctx, currentUpdateAt)
	if err != nil {
		return nil, err
	}

	return nextBank, nil
}

func (s *service) PreviousPage(ctx context.Context, currentUpdateAt time.Time) (*Bank, error) {
	prevBank, err := s.store.getPreviousPageBankByID(ctx, currentUpdateAt)
	if err != nil {
		return nil, err
	}

	return prevBank, nil
}

func main() {
	// Initialize your database connection and store
	db, err := sql.Open("your-database-driver", "your-database-url")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	router := mux.NewRouter()
	store := &store{db: db} // Initialize your store

	service := &service{store: store}

	router.HandleFunc("/banks/next", NextPageHandler(service)).Methods("GET")
	router.HandleFunc("/banks/previous", PreviousPageHandler(service)).Methods("GET")

	http.Handle("/", router)
	http.ListenAndServe(":8080", nil)
}

// Define your HTTP handlers here

func NextPageHandler(s *service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUpdateAt, err := getCurrentUpdateAtFromRequest(r)
		if err != nil {
			responseError(w, http.StatusBadRequest, errors.New("invalid timestamp"))
			return
		}

		nextBank, err := s.NextPage(r.Context(), currentUpdateAt)
		if err != nil {
			responseError(w, http.StatusInternalServerError, err)
			return
		}

		responseJSON(w, http.StatusOK, nextBank)
	}
}

func PreviousPageHandler(s *service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUpdateAt, err := getCurrentUpdateAtFromRequest(r)
		if err != nil {
			responseError(w, http.StatusBadRequest, errors.New("invalid timestamp"))
			return
		}

		prevBank, err := s.PreviousPage(r.Context(), currentUpdateAt)
		if err != nil {
			responseError(w, http.StatusInternalServerError, err)
			return
		}

		responseJSON(w, http.StatusOK, prevBank)
	}
}

// Utility functions

func getCurrentUpdateAtFromRequest(r *http.Request) (time.Time, error) {
	currentUpdateAtStr := r.URL.Query().Get("current_update_at")
	if currentUpdateAtStr == "" {
		return time.Time{}, errors.New("missing current_update_at parameter")
	}

	currentUpdateAt, err := time.Parse(time.RFC3339, currentUpdateAtStr)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to parse current_update_at")
	}

	return currentUpdateAt, nil
}

func responseJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func responseError(w http.ResponseWriter, statusCode int, err error) {
	responseJSON(w, statusCode, map[string]string{"error": err.Error()})
}
