package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

type UpdateKDSStateInput struct {
	TenantID uuid.UUID
	OrderID  uuid.UUID
	State    domain.KDSState
}

type UpdateKDSStateUseCase struct {
	DB        *postgres.DB
	KDS       domain.KDSRepository
	Orders    domain.OrderRepository
	Events    domain.EventRepository
	Publisher domain.EventPublisher
	Clock     func() time.Time
}

func NewUpdateKDSStateUseCase(
	db *postgres.DB,
	kds domain.KDSRepository,
	orders domain.OrderRepository,
	events domain.EventRepository,
	publisher domain.EventPublisher,
) *UpdateKDSStateUseCase {
	return &UpdateKDSStateUseCase{
		DB:        db,
		KDS:       kds,
		Orders:    orders,
		Events:    events,
		Publisher: publisher,
		Clock:     time.Now,
	}
}

func (uc *UpdateKDSStateUseCase) Execute(
	ctx context.Context,
	input UpdateKDSStateInput,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if input.TenantID == uuid.Nil {
		return domain.ErrInvalidTenantID
	}

	if input.OrderID == uuid.Nil {
		return domain.ErrInvalidOrderID
	}

	if !input.State.Valid() {
		return domain.ErrInvalidKDSState
	}

	tx, err := uc.DB.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin kds state transaction: %w", err)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()

	kdsOrder, err := uc.KDS.GetByOrderID(
		ctx,
		tx,
		input.TenantID,
		input.OrderID,
	)
	if err != nil {
		return fmt.Errorf("get kds order: %w", err)
	}

	previousState := kdsOrder.State

	if err := kdsOrder.TransitionTo(
		input.State,
		uc.Clock().UTC(),
	); err != nil {
		return fmt.Errorf("kds transition: %w", err)
	}

	if err := uc.KDS.UpdateState(
		ctx,
		tx,
		input.TenantID,
		input.OrderID,
		previousState,
		input.State,
	); err != nil {
		return fmt.Errorf("update kds state: %w", err)
	}

	// Keep the order aggregate and KDS aggregate synchronized.
	orderState := domain.OrderState(input.State)

	if err := uc.Orders.UpdateState(
		ctx,
		tx,
		input.TenantID,
		input.OrderID,
		domain.OrderState(previousState),
		orderState,
	); err != nil {
		return fmt.Errorf("update order state: %w", err)
	}

	event := domain.KDSStateUpdatedEvent{
		EventID:       uuid.New(),
		TenantID:      input.TenantID,
		OrderID:       input.OrderID,
		PreviousState: previousState,
		State:         input.State,
		OccurredAtT:   uc.Clock().UTC(),
	}

	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return fmt.Errorf("append kds event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit kds transaction: %w", err)
	}

	committed = true

	// This is an immediate best-effort notification.
	// The durable event already exists in the outbox.
	if err := uc.Publisher.Publish(ctx, event); err != nil {
		return fmt.Errorf("publish kds event: %w", err)
	}

	return nil
}
