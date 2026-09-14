package domain

import (
	"time"

	"github.com/google/uuid"
)

type KDSState string

const (
	KDSStatePending   KDSState = "pending"
	KDSStatePreparing KDSState = "preparing"
	KDSStateReady     KDSState = "ready"
	KDSStateCompleted KDSState = "completed"
)

func (s KDSState) Valid() bool {
	switch s {
	case KDSStatePending,
		KDSStatePreparing,
		KDSStateReady,
		KDSStateCompleted:
		return true
	default:
		return false
	}
}

func (s KDSState) CanTransitionTo(next KDSState) bool {
	switch s {
	case KDSStatePending:
		return next == KDSStatePreparing

	case KDSStatePreparing:
		return next == KDSStateReady

	case KDSStateReady:
		return next == KDSStateCompleted

	case KDSStateCompleted:
		return false

	default:
		return false
	}
}

type KDSOrder struct {
	ID        uuid.UUID
	TenantID  TenantID
	OrderID   OrderID
	State     KDSState
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewKDSOrder(
	tenantID uuid.UUID,
	orderID OrderID,
	now time.Time,
) (*KDSOrder, error) {
	tid, err := NewTenantID(tenantID)
	if err != nil {
		return nil, err
	}

	if orderID.UUID() == uuid.Nil {
		return nil, ErrInvalidOrderID
	}

	return &KDSOrder{
		ID:        uuid.New(),
		TenantID:  tid,
		OrderID:   orderID,
		State:     KDSStatePending,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (k *KDSOrder) TransitionTo(next KDSState, now time.Time) error {
	if !next.Valid() {
		return ErrInvalidKDSState
	}

	if !k.State.CanTransitionTo(next) {
		return ErrInvalidStateTransition
	}

	k.State = next
	k.UpdatedAt = now

	return nil
}
