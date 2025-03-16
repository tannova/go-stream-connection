package connection

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type StreamServer interface {
	Send(*anypb.Any) error
	Context() context.Context
}

func NewGRPCStream(connKey any, conn StreamServer, ping bool, metadata map[string]string, messSize int, timePing time.Duration) Connection {
	newConn := &grpcStream{
		connID:           uuid.NewString(),
		connKey:          connKey,
		metadata:         metadata,
		conn:             conn,
		establishedAt:    time.Now(),
		outgoingMessages: make(chan *outgoingMessage, messSize),
		timePing:         timePing,
	}

	go newConn.handleStream(ping)

	return newConn
}

type grpcStream struct {
	connID             string
	connKey            any
	metadata           map[string]string
	establishedAt      time.Time
	conn               StreamServer
	outgoingMessages   chan *outgoingMessage
	disconnectHandlers []DisconnectHandler
	timePing           time.Duration

	closed bool
	mutex  sync.Mutex
}

// ID returns the ID of the connection.
//
// No parameters.
// Returns a string.
func (c *grpcStream) ID() string {
	return c.connID
}

// ConnKey returns the ConnKey associated with the connection.
//
// No parameters.
// Returns a any.
func (c *grpcStream) ConnKey() any {
	return c.connKey
}

// Metadata returns the metadata associated with the connection.
//
// No parameters.
// Returns a map[string]string.
func (c *grpcStream) Metadata() map[string]string {
	return c.metadata
}

// EstablishedAt returns the time at which the connection was established.
//
// No parameters.
// Returns a time.Time object.
func (c *grpcStream) EstablishedAt() time.Time {
	return c.establishedAt
}

// Send sends a message over the connection.
//
// ctx: The context for the message.
// message: The message to be sent.
func (c *grpcStream) Send(ctx context.Context, message any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return
	}

	protoMessage, ok := message.(proto.Message)
	if !ok {
		//logging.Logger(ctx).Error("invalid message",
		//	zap.String("conn_id", c.connID),
		//	zap.Int64("player_id", c.playerID),
		//	zap.Any("message", message))
		return
	}

	payload, err := anypb.New(protoMessage)
	if err != nil {
		//logging.Logger(ctx).Error("create any message failed",
		//	zap.String("conn_id", c.connID),
		//	zap.Int64("player_id", c.playerID),
		//	zap.Any("message", message),
		//	zap.Error(err))
		return
	}

	if len(c.outgoingMessages) == cap(c.outgoingMessages) {
		//logging.Logger(ctx).Error("outgoing message queue is full",
		//	zap.String("conn_id", c.connID),
		//	zap.Int64("player_id", c.playerID))
		return
	}

	c.outgoingMessages <- &outgoingMessage{
		ctx:     ctx,
		payload: payload,
	}
}

// SendGoingAway sends a close message of type CloseGoingAway to the connection.
//
// ctx: The context.Context object that represents the current context.
// Return: None.
func (c *grpcStream) SendGoingAway(ctx context.Context) {
	if c.closed {
		return
	}
	c.Close(ctx)
}

// OnDisconnect registers one or more handler functions to be called when the connection is closed.
//
// handlers: The disconnect handlers to be appended.
func (c *grpcStream) OnDisconnect(handlers ...DisconnectHandler) {
	c.mutex.Lock()

	if c.closed {
		c.mutex.Unlock()

		for _, handler := range handlers {
			ctx := context.TODO()
			handler(ctx, c.connKey, c.connID)
		}

		return
	}

	c.disconnectHandlers = append(c.disconnectHandlers, handlers...)
	c.mutex.Unlock()
}

// OnReceive registers one or more handler functions to be called when a message is received.
//
// handlers: one or more ReceiveHandlers to be appended to the receiveHandlers slice.
func (c *grpcStream) OnReceive(handlers ...ReceiveHandler) {
	// grpc-stream only support server streaming
}

// Close closes the connection.
//
// ctx - The context to use for the operation.
// No return value.
func (c *grpcStream) Close(ctx context.Context) {
	c.mutex.Lock()
	if c.closed {
		c.mutex.Unlock()
		return
	}
	c.closed = true
	c.mutex.Unlock()

	close(c.outgoingMessages)
	for _, handler := range c.disconnectHandlers {
		handler(ctx, c.connKey, c.connID)
	}
}

type outgoingMessage struct {
	ctx     context.Context
	payload *anypb.Any
}

func (c *grpcStream) handleStream(ping bool) {
	stopPing := c.startPing(ping)
	defer func() {
		stopPing()
		c.Close(context.TODO())
	}()

	for {
		select {
		case message, ok := <-c.outgoingMessages:
			if !ok {
				return
			}

			if err := c.conn.Send(message.payload); err != nil {
				if status.Code(err) != codes.Canceled {
					//logging.Logger(message.ctx).Error("write message failed",
					//	zap.String("conn_id", c.connID),
					//	zap.Int64("player_id", c.playerID),
					//	zap.Any("payload", message.payload),
					//	zap.Error(err))
				}
				return
			}

		case <-c.conn.Context().Done():
			return
		}
	}
}

func (c *grpcStream) startPing(ping bool) func() {
	if !ping {
		return func() {}
	}

	ctx := context.TODO()
	ticker := time.NewTicker(c.timePing)
	stop := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				c.Send(ctx, &emptypb.Empty{})
			case <-stop:
				return
			}
		}
	}()

	return func() {
		ticker.Stop()
		close(stop)
	}
}
