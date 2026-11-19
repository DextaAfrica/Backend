package request

type RecordAuditRequest struct {
	ActorID    string         `json:"actorId"`
	ActorEmail string         `json:"actorEmail"`
	Action     string         `json:"action" validate:"required"`
	Entity     string         `json:"entity" validate:"required"`
	EntityID   string         `json:"entityId"`
	IPAddress  string         `json:"ipAddress"`
	UserAgent  string         `json:"userAgent"`
	Metadata   map[string]any `json:"metadata"`
}
