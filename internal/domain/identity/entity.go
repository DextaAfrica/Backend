package identity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string

const (
	RoleSuperAdmin         Role = "SUPER_ADMIN"
	RoleOperationsDirector Role = "OPERATIONS_DIRECTOR"
	RolePropertyManager    Role = "PROPERTY_MANAGER"
	RoleSalesLead          Role = "SALES_LEAD"
	RoleRealtorAgent       Role = "REALTOR_AGENT"
	RoleFinanceOfficer     Role = "FINANCE_OFFICER"
	RoleLegalCompliance    Role = "LEGAL_COMPLIANCE"
	RoleClient             Role = "CLIENT"
)

type UserStatus string

const (
	StatusActive              UserStatus = "ACTIVE"
	StatusSuspended           UserStatus = "SUSPENDED"
	StatusPendingVerification UserStatus = "PENDING_VERIFICATION"
)

// User represents internal personnel, partners, realtors, or clients.
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	FirstName    string         `gorm:"type:varchar(100);not null" json:"first_name"`
	LastName     string         `gorm:"type:varchar(100);not null" json:"last_name"`
	PhoneNumber  string         `gorm:"type:varchar(50);index" json:"phone_number"`
	Role         Role           `gorm:"type:varchar(50);not null;default:'CLIENT';index" json:"role"`
	Status       UserStatus     `gorm:"type:varchar(50);not null;default:'ACTIVE';index" json:"status"`
	CountryCode  string         `gorm:"type:varchar(10);default:'NG'" json:"country_code"`
	AvatarURL    string         `gorm:"type:text" json:"avatar_url"`
	LastLoginAt  *time.Time     `json:"last_login_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// RefreshToken stores rotating refresh tokens with device bindings.
type RefreshToken struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID            uuid.UUID  `gorm:"type:uuid;index;not null" json:"user_id"`
	TokenHash         string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"`
	DeviceFingerprint string     `gorm:"type:varchar(255)" json:"device_fingerprint"`
	IPAddress         string     `gorm:"type:varchar(64)" json:"ip_address"`
	ExpiresAt         time.Time  `gorm:"not null" json:"expires_at"`
	RevokedAt         *time.Time `json:"revoked_at"`
	CreatedAt         time.Time  `json:"created_at"`
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// AuditLog provides immutable audit trails for compliance.
type AuditLog struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ActorID      uuid.UUID `gorm:"type:uuid;index" json:"actor_id"`
	ActorEmail   string    `gorm:"type:varchar(255)" json:"actor_email"`
	ActorRole    string    `gorm:"type:varchar(50)" json:"actor_role"`
	Action       string    `gorm:"type:varchar(100);index;not null" json:"action"`
	ResourceType string    `gorm:"type:varchar(100);index;not null" json:"resource_type"`
	ResourceID   string    `gorm:"type:varchar(255);index" json:"resource_id"`
	IPAddress    string    `gorm:"type:varchar(64)" json:"ip_address"`
	UserAgent    string    `gorm:"type:text" json:"user_agent"`
	DiffBefore   string    `gorm:"type:jsonb" json:"diff_before,omitempty"`
	DiffAfter    string    `gorm:"type:jsonb" json:"diff_after,omitempty"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

