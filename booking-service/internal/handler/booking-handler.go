package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// flightpb "flight-booking/.gen/.proto/flight"
	"flight-booking/booking-service/internal/model"
	"flight-booking/booking-service/internal/service"
	"flight-booking/booking-service/pkg/grpcclient"
)

type BookingHandler struct {
	service *service.BookingService
	flightClient *grpcclient.FlightClient
}

func NewBookingHandler(s *service.BookingService) *BookingHandler {
	return &BookingHandler{service: s}
}

func (h *BookingHandler) SearchFlights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	origin := r.URL.Query().Get("origin")
	destination := r.URL.Query().Get("destination")
	dateStr := r.URL.Query().Get("date")

	if origin == "" || destination == "" {
		http.Error(w, "origin and destination are required", http.StatusBadRequest)
		return
	}

	fc := grpcclient.NewFlightClient()
	defer fc.Close()

	var date *time.Time
	if dateStr != "" {
		parsed, err := time.Parse("2005-02-21", dateStr)
		if err != nil {
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
		}
		date = &parsed
	}

	resp, err := fc.SearchFlights(
		r.Context(),
		origin,
		destination,
		date,
	)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.NotFound {
			http.Error(w, "No flights found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
 	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *BookingHandler) GetFlight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/flights/")
	if id == "" {
		http.Error(w, "Flight ID is required", http.StatusBadRequest)
		return
	}

	fc := grpcclient.NewFlightClient()
	defer fc.Close()

	resp, err := fc.GetFlight(r.Context(), id)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.NotFound {
			http.Error(w, "flight not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	booking, err := h.service.CreateBooking(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(booking)
}

func (h *BookingHandler) GetBooking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/bookings/")
	if id == "" {
		http.Error(w, "booking ID is required", http.StatusBadRequest)
		return
	}

	booking, err := h.service.GetByID(r.Context(), id)
	if err != nil || booking == nil {
		http.Error(w, "Booking not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(booking)
}

func (h *BookingHandler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/cancel"), "/bookings/")
	if id == "" {
		http.Error(w, "booking ID is required", http.StatusBadRequest)
		return
	}

	if err := h.service.CancelBooking(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BookingHandler) ListBookings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	bookings, err := h.service.GetByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookings)
}
