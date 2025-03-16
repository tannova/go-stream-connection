package connection

import (
	"context"
	"sync"
)

type Hub interface {
	// AddConnection adds a new connection to the hub, associating it with a player ID.
	AddConnection(ctx context.Context, conn Connection)

	// Send sends a message to a specific player in the hub.
	Send(ctx context.Context, connKey any, message any)
	// SendGoingAway sends a going away message to a specific player in the hub.
	SendGoingAway(ctx context.Context, connKey any)

	// Broadcast sends a message to all players in the hub.
	Broadcast(ctx context.Context, message any)
	// BroadcastGoingAway sends a going away message to all players in the hub.
	BroadcastGoingAway(ctx context.Context)

	// OnConnect registers a new connection handler to be called when a new connection is added to the hub.
	OnConnect(handlers ...NewConnectionHandler)
	// OnDisconnect registers a new disconnection handler to be called when a connection is removed from the hub.
	OnDisconnect(handlers ...DisconnectHandler)
	// OnReceive registers a new receive handler to be called when a message is received from a connection in the hub.
	OnReceive(handlers ...ReceiveHandler)

	// GetMembers returns a list of player IDs in the hub.
	GetMembers() []any
	// GetConnections returns a list of connections associated with a player ID in the hub.
	GetConnections(connKey any) []Connection
	// GetConnectionByID returns a connection object based on its ID in the hub.
	GetConnectionByID(ctx context.Context, connID string) Connection

	GetTotalConnections() int
}

func NewHub() Hub {
	return &hub{
		connections:    map[string]Connection{},
		anyConnections: map[any][]Connection{},
	}
}

type hub struct {
	connections    map[string]Connection // conn_id to Connection
	anyConnections map[any][]Connection  // player_id to Connections

	newConnectionHandlers []NewConnectionHandler
	disconnectHandlers    []DisconnectHandler
	receiveHandlers       []ReceiveHandler

	mutex sync.RWMutex

	totalConnections     int
	gameTotalConnections map[string]int
}

// AddConnection adds a new connection to the Hub.
//
// Parameters:
//   - ctx: The context.Context object.
//   - connKey: The ID of the player.
//   - conn: The web socket connection.
func (h *hub) AddConnection(ctx context.Context, conn Connection) {
	h.mutex.Lock()

	h.connections[conn.ID()] = conn
	playerConnections := h.anyConnections[conn.ConnKey()]
	h.anyConnections[conn.ConnKey()] = append(playerConnections, conn)

	//stats
	h.totalConnections++

	h.mutex.Unlock()

	conn.OnDisconnect(h.removeConnection)
	conn.OnDisconnect(h.disconnectHandlers...)
	conn.OnReceive(h.receiveHandlers...)

	for _, handler := range h.newConnectionHandlers {
		handler(ctx, conn.ConnKey(), conn)
	}
}

// Send sends a message to a specific player in the Hub.
//
// Parameters:
//   - ctx: The context.Context object.
//   - connKey: The ID of the player.
//   - message: The message to send.
func (h *hub) Send(ctx context.Context, connKey any, message any) {
	for _, conn := range h.GetConnections(connKey) {
		if conn == nil {
			continue
		}
		conn.Send(ctx, message)
	}
}

// SendGoingAway sends a "going away" message to a specific player in the Hub.
// It means telling client to disconnect and close the connection.
//
// Parameters:
//   - ctx: The context.Context object.
//   - connKey: The ID of the player.
func (h *hub) SendGoingAway(ctx context.Context, connKey any) {
	for _, conn := range h.GetConnections(connKey) {
		if conn == nil {
			continue
		}
		conn.SendGoingAway(ctx)
	}
}

// Broadcast sends a message to all players in the Hub.
//
// Parameters:
//   - ctx: The context.Context object.
//   - message: The message to send.
func (h *hub) Broadcast(ctx context.Context, message any) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for _, conn := range h.connections {
		go conn.Send(ctx, message)
	}
}

// BroadcastGoingAway sends a "going away" message to all players in the Hub.
// It means telling all clients to disconnect and close the connections.
//
// Parameters:
//   - ctx: The context.Context object.
func (h *hub) BroadcastGoingAway(ctx context.Context) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for _, conn := range h.connections {
		go conn.SendGoingAway(ctx)
	}
}

// OnConnect registers a new connection handler to be called when a new connection is added to the Hub.
//
// Parameters:
//   - handlers: The connection handlers.
func (h *hub) OnConnect(handlers ...NewConnectionHandler) {
	h.newConnectionHandlers = append(h.newConnectionHandlers, handlers...)
}

// OnDisconnect registers a new disconnection handler to be called when a connection is removed from the Hub.
//
// Parameters:
//   - handlers: The disconnection handlers.
func (h *hub) OnDisconnect(handlers ...DisconnectHandler) {
	h.disconnectHandlers = append(h.disconnectHandlers, handlers...)
}

// OnReceive registers a new receive handler to be called when a message is received in the Hub.
//
// Parameters:
//   - handlers: The receive handlers.
func (h *hub) OnReceive(handlers ...ReceiveHandler) {
	h.receiveHandlers = append(h.receiveHandlers, handlers...)
}

// GetMembers returns a list of player IDs in the Hub.
//
// Parameters:
//   - ctx: The context.Context object.
//
// Returns:
//   - []string: The list of player IDs.
func (h *hub) GetMembers() []any {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	var connKeys []any
	for connKey := range h.anyConnections {
		connKeys = append(connKeys, connKey)
	}

	return connKeys
}

// GetConnections returns a list of connections for a specific player in the Hub.
//
// Parameters:
//   - ctx: The context.Context object.
//   - connKey: The ID of the player.
//
// Returns:
//   - []Connection: The list of connections.
func (h *hub) GetConnections(connKey any) []Connection {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return h.anyConnections[connKey]
}

// GetConnectionByID returns a specific connection in the Hub based on its ID.
//
// Parameters:
//   - ctx: The context.Context object.
//   - connID: The ID of the connection.
//
// Returns:
//   - Connection: The connection object.
func (h *hub) GetConnectionByID(ctx context.Context, connID string) Connection {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return h.connections[connID]
}

func (h *hub) removeConnection(ctx context.Context, connKey any, connID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	delete(h.connections, connID)
	var connIndex = -1
	for i, conn := range h.anyConnections[connKey] {
		if conn.ID() == connID {
			connIndex = i
			break
		}
	}

	if connIndex >= 0 {
		h.anyConnections[connKey] = append(h.anyConnections[connKey][:connIndex], h.anyConnections[connKey][connIndex+1:]...)
	}
	if len(h.anyConnections[connKey]) == 0 {
		delete(h.anyConnections, connKey)
	}

}

func (h *hub) GetTotalConnections() int {
	return h.totalConnections
}

func (h *hub) GetGameTotalConnections(gameID string) int {
	return h.gameTotalConnections[gameID]
}
