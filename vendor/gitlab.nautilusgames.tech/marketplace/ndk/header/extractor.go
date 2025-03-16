package header

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc/metadata"
)

const (
	PlayerID          = "x-player-id"
	TenantID          = "x-tenant-id"
	ApiKey            = "x-api-key"
	TenantPlayerID    = "x-tenant-player-id"
	GameID            = "x-game-id"
	Currency          = "x-currency"
	TenantPlayerToken = "x-tenant-player-token"
	AgentID           = "x-agent-id"
	// used for proxy
	XHost  = "x-host"
	XProto = "x-proto"
)

type Extractor interface {
	Get(ctx context.Context, name string) []string
	GetFirst(ctx context.Context, name string) string
	GetPlayerID(ctx context.Context) (int64, error)
	GetTenantID(ctx context.Context) (string, bool)
	GetTenantPlayerID(ctx context.Context) (string, bool)
	GetCurrency(ctx context.Context) (string, bool)
	GetAgentID(ctx context.Context) (string, bool)
	GetGameID(ctx context.Context) (string, bool)
}

type extractor struct {
}

func New() Extractor {
	return &extractor{}
}

func (t *extractor) Get(ctx context.Context, name string) []string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}

	return md.Get(name)
}

func (t *extractor) GetFirst(ctx context.Context, name string) string {
	values := t.Get(ctx, name)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (t *extractor) GetPlayerID(ctx context.Context) (int64, error) {
	values := t.Get(ctx, PlayerID)
	if len(values) == 0 {
		return 0, fmt.Errorf("%s not found", PlayerID)
	}

	playerID, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %v", PlayerID, err)
	}

	return playerID, nil
}

func (t *extractor) GetTenantID(ctx context.Context) (string, bool) {
	values := t.Get(ctx, TenantID)
	if len(values) == 0 {
		return "", false
	}

	return values[0], true
}

func (t *extractor) GetTenantPlayerID(ctx context.Context) (string, bool) {
	values := t.Get(ctx, TenantPlayerID)
	if len(values) == 0 {
		return "", false
	}

	return values[0], true
}

func (t *extractor) GetCurrency(ctx context.Context) (string, bool) {
	values := t.Get(ctx, Currency)
	if len(values) == 0 {
		return "", false
	}

	return values[0], true
}

func (t *extractor) GetAgentID(ctx context.Context) (string, bool) {
	values := t.Get(ctx, AgentID)
	if len(values) == 0 {
		return "", false
	}

	return values[0], true
}

func (t *extractor) GetGameID(ctx context.Context) (string, bool) {
	values := t.Get(ctx, GameID)
	if len(values) == 0 {
		return "", false
	}

	return values[0], true
}
