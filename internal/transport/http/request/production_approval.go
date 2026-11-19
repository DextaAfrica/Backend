package request

type CreateProductionReleaseRequest struct {
	Title         string  `json:"title" validate:"required,max=255"`
	TargetSystem  string  `json:"targetSystem" validate:"required,max=128"`
	TicketNumber  string  `json:"ticketNumber" validate:"omitempty,max=64"`
	RiskLevel     string  `json:"riskLevel" validate:"omitempty,oneof=LOW MEDIUM HIGH CRITICAL"`
	Justification string  `json:"justification" validate:"required"`
	RollBackPlan  string  `json:"rollbackPlan" validate:"required"`
	DiffSummary   *string `json:"diffSummary"`
}

type DecideProductionReleaseRequest struct {
	Action string `json:"action" validate:"required,oneof=APPROVE REJECT"`
	Notes  string `json:"notes" validate:"omitempty,max=1000"`
}
