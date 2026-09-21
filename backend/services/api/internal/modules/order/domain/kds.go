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

type KDSOrder struct {
	ID           uuid.UUID
	TenantID     TenantID
	RestaurantID RestaurantID
	OrderID      OrderID
	State        KDSState
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewKDSOrder(
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	orderID OrderID,
	now time.Time,
) (*KDSOrder, error) {
	tid, err := NewTenantID(tenantID)
	if err != nil {
		return nil, err
	}

	rid, err := NewRestaurantID(restaurantID)
	if err != nil {
		return nil, err
	}

	return &KDSOrder{
		ID:           uuid.New(),
		TenantID:     tid,
		RestaurantID: rid,
		OrderID:      orderID,
		State:        KDSStatePending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (k *KDSOrder) TransitionTo(next KDSState, now time.Time) error {
	if !next.Valid() {
		return ErrInvalidKDSState
	}

	switch k.State {
	case KDSStatePending:
		if next != KDSStatePreparing {
			return ErrInvalidStateTransition
		}

	case KDSStatePreparing:
		if next != KDSStateReady {
			return ErrInvalidStateTransition
		}

	case KDSStateReady:
		if next != KDSStateCompleted {
			return ErrInvalidStateTransition
		}

	case KDSStateCompleted:
		return ErrInvalidStateTransition

	default:
		return ErrInvalidKDSState
	}

	k.State = next
	k.UpdatedAt = now

	return nil
}
