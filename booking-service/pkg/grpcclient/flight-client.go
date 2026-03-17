package grpcclient

import (
    "context"
    "log"
    "os"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/protobuf/types/known/timestamppb"

    flightpb "flight-booking/.gen/.proto/flight"
)

type FlightClient struct {
    conn *grpc.ClientConn
    client flightpb.FlightServiceClient
}

func NewFlightClient() *FlightClient {
    addr := os.Getenv("FLIGHT_SERVICE_ADDR")
    if addr == "" {
        addr = "flight-service:9090"
    }

    conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect to Flight Service: %v", err)
    }

    return &FlightClient{
        conn: conn,
        client: flightpb.NewFlightServiceClient(conn),
    }
}

func (c *FlightClient) Close() error {
    return c.conn.Close()
}

func (c *FlightClient) GetFlight(ctx context.Context, id string) (*flightpb.Flight, error) {
    resp, err := c.client.GetFlight(ctx, &flightpb.GetFlightRequest{Id: id})
    if err != nil {
        return nil, err
    }
    return resp.Flight, nil
}

func (c *FlightClient) SearchFlights(ctx context.Context, origin, destination string, date *time.Time) ([]*flightpb.Flight, error) {
    var datePb *timestamppb.Timestamp
    if date != nil {
        datePb = timestamppb.New(*date)
    }

    resp, err := c.client.SearchFlights(
        ctx,
        &flightpb.SearchFlightsRequest{
            Origin: origin,
            Destination: destination,
            Date: datePb,
        },
    )
    if err != nil {
        return nil, err
    }
    return resp.Flights, nil
}

func (c *FlightClient) ReserveSeats(ctx context.Context, flightID string, seatCount int32, bookingID string) (string, error) {
    resp, err := c.client.ReserveSeats(ctx, &flightpb.ReserveSeatsRequest{
        FlightId:  flightID,
        SeatCount: seatCount,
        BookingId: bookingID,
    })
    if err != nil {
        return "", err
    }

    return resp.ReservationId, nil
}

func (c *FlightClient) ReleaseReservation(ctx context.Context, bookingID string) error {
    _, err := c.client.ReleaseReservation(ctx, &flightpb.ReleaseReservationRequest{BookingId: bookingID})
    return err
}
