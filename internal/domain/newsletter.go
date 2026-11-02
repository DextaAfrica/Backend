package domain

import (
	"context"
	"time"
)

type SubscriberStatus string

const (
	SubscriberStatusSubscribed   SubscriberStatus = "subscribed"
	SubscriberStatusUnsubscribed SubscriberStatus = "unsubscribed"
)

type Subscriber struct {
	ID             string
	Email          string
	Status         SubscriberStatus
	SubscribedAt   time.Time
	UnsubscribedAt *time.Time
	CreatedAt      time.Time
}

type SubscriberRepository interface {
	Upsert(ctx context.Context, email string) (*Subscriber, error)
	Unsubscribe(ctx context.Context, email string) error
	GetByEmail(ctx context.Context, email string) (*Subscriber, error)
	List(ctx context.Context, params ListParams) (PageResult[Subscriber], error)
}
