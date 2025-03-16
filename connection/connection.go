package connection

import (
	"context"
	"time"
)

type Type string

type NewConnectionHandler func(ctx context.Context, connKey any, connection Connection)

type DisconnectHandler func(ctx context.Context, connKey any, connID string)

type ReceiveHandler func(ctx context.Context, connKey any, connID string, message []byte)

type Connection interface {
	// ID returns the connection ID.
	ID() string
	// Metadata returns the metadata associated with the connection.
	Metadata() map[string]string
	// EstablishedAt returns the time when the connection was established.
	EstablishedAt() time.Time
	// Send sends a message to the connected client.
	Send(ctx context.Context, message any)
	// SendGoingAway sends a "going away" message to the connected client.
	SendGoingAway(ctx context.Context)
	// OnDisconnect registers one or more handler functions to be called when the connection is closed.
	OnDisconnect(handlers ...DisconnectHandler)
	// OnReceive registers one or more handler functions to be called when a message is received.
	OnReceive(handlers ...ReceiveHandler)
	// Close closes the connection.
	Close(ctx context.Context)
}
