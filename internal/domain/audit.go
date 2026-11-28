package domain

import (
	"context"
	"time"
)

type AuditLog struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actorId,omitempty"`
	ActorEmail string         `json:"actorEmail,omitempty"`
	Action     string         `json:"action"`
	Entity     string         `json:"entity"`
	EntityID   string         `json:"entityId,omitempty"`
	IPAddress  string         `json:"ipAddress,omitempty"`
	UserAgent  string         `json:"userAgent,omitempty"`
	Metadata   map[string]any `json:"metadata"`
	PrevHash   string         `json:"prevHash"`
	RecordHash string         `json:"recordHash"`
	CreatedAt  time.Time      `json:"createdAt"`
}

type AuditRepository interface {
	Record(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, params ListParams) (PageResult[AuditLog], error)
	VerifyChain(ctx context.Context) (bool, error)
}
