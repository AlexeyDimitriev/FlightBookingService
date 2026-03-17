package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"flight-booking/flight-service/internal/model"
)

type FlightRepo struct {
 	db *sql.DB
}

func NewFlightRepo(db *sql.DB) *FlightRepo {
	return &FlightRepo{db: db}
}

func (r *FlightRepo) Search(ctx context.Context, origin string, destination string, date *time.Time) ([]model.Flight, error) {
	query := `
		SELECT id, flight_number, departure_date, airline, origin_airport, 
		destination_airport, departure_time, arrival_time, total_seats, 
		available_seats, price, status 
		FROM flights 
		WHERE origin_airport = $1 AND destination_airport = $2 
		AND status = 'SCHEDULED'
	`

	args := []interface{}{origin, destination}
	if date != nil {
		query += " AND departure_date = $3"
		args = append(args, date)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flights []model.Flight
	for rows.Next() {
		var f model.Flight
		err := rows.Scan(&f.ID, &f.FlightNumber, &f.DepartureDate, &f.Airline,
		&f.OriginAirport, &f.DestinationAirport, &f.DepartureTime,
		&f.ArrivalTime, &f.TotalSeats, &f.AvailableSeats, &f.Price, &f.Status)
		if err != nil {
			return nil, err
		}
		flights = append(flights, f)
	}
	return flights, rows.Err()
}

func (r *FlightRepo) GetByID(ctx context.Context, id string) (*model.Flight, error) {
	var f model.Flight
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, flight_number, departure_date, airline, origin_airport,
		destination_airport, departure_time, arrival_time, total_seats,
		available_seats, price, status 
		FROM flights WHERE id = $1`,
		id,
	).Scan(
		&f.ID, &f.FlightNumber, &f.DepartureDate, &f.Airline,
		&f.OriginAirport, &f.DestinationAirport, &f.DepartureTime,
		&f.ArrivalTime, &f.TotalSeats, &f.AvailableSeats, &f.Price, &f.Status,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &f, err
}

func (r *FlightRepo) ReserveSeats(ctx context.Context, tx *sql.Tx, flightID string, seatCount int32, bookingID string) (string, error) {
	var available int32
	err := tx.QueryRowContext(
		ctx, 
		"SELECT available_seats FROM flights WHERE id = $1 FOR UPDATE", 
		flightID,
	).Scan(
		&available,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("Flight not found")
		}
		return "", err
	}

	if available < seatCount {
		return "", fmt.Errorf("Not enough seats")
	}

	_, err = tx.ExecContext(
		ctx, 
		"UPDATE flights SET available_seats = available_seats - $1, updated_at = NOW() WHERE id = $2", 
		seatCount, flightID,
	)
	if err != nil {
		return "", err
	}

	var reservationID string
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO seat_reservations (flight_id, booking_id, seat_count, status)
		VALUES ($1, $2, $3, 'ACTIVE')
		RETURNING id`, 
		flightID, bookingID, seatCount,
	).Scan(
		&reservationID,
	)

	return reservationID, err
}

func (r *FlightRepo) ReleaseReservation(ctx context.Context, tx *sql.Tx, bookingID string) (bool, error) {
	var reservationID, flightID string
	var seatCount int32

	err := tx.QueryRowContext(
		ctx,
		`SELECT id, flight_id, seat_count FROM seat_reservations 
		WHERE booking_id = $1 AND status = 'ACTIVE' FOR UPDATE`, 
		bookingID,
	).Scan(
		&reservationID,
		&flightID,
		&seatCount,
	)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	_, err = tx.ExecContext(
		ctx, 
		"UPDATE flights SET available_seats = available_seats + $1 WHERE id = $2", 
		seatCount, flightID,
	)
	if err != nil {
		return false, err
	}

	_, err = tx.ExecContext(
		ctx, 
		"UPDATE seat_reservations SET status = 'RELEASED', updated_at = NOW() WHERE id = $1", 
		reservationID,
	)
	return err == nil, err
}
