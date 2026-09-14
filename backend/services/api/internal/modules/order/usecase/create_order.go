package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"restaurantflow/internal/modules/order/domain"
	"restaurantflow/internal/platform/postgres"
)

type CreateOrderInput struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	Currency   string
	Items      []CreateOrderItemInput
}

type CreateOrderItemInput struct {
	ProductID uuid.UUID
	Name      string
	Quantity  int32
	UnitPrice int64
}

type CreateOrderUseCase struct {
	DB        *postgres.DB
	Orders    domain.OrderRepository
	KDS       domain.KDSRepository
	Inventory domain.InventoryRepository
	Events    domain.EventRepository
	Clock     func() time.Time
}

func NewCreateOrderUseCase(
	db *postgres.DB,
	orders domain.OrderRepository,
	kds domain.KDSRepository,
	inventory domain.InventoryRepository,
	events domain.EventRepository,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		DB:        db,
		Orders:    orders,
		KDS:       kds,
		Inventory: inventory,
		Events:    events,
		Clock:     time.Now,
	}
}

func (uc *CreateOrderUseCase) Execute(
	ctx context.Context,
	input CreateOrderInput,
) (*domain.Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if input.CustomerID == uuid.Nil {
		return nil, domain.ErrInvalidCustomerID
	}

	if len(input.Items) == 0 {
		return nil, domain.ErrInvalidOrderItems
	}

	currency := input.Currency

	items := make([]domain.OrderItem, 0, len(input.Items))

	for _, itemInput := range input.Items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		money, err := domain.NewMoney(
			itemInput.UnitPrice,
			currency,
		)
		if err != nil {
			return nil, fmt.Errorf("create order item price: %w", err)
		}

		item, err := domain.NewOrderItem(
			itemInput.ProductID,
			itemInput.Name,
			itemInput.Quantity,
			money,
		)
		if err != nil {
			return nil, fmt.Errorf("create order item: %w", err)
		}

		items = append(items, item)
	}

	now := uc.Clock().UTC()

	order, err := domain.NewOrder(
		input.TenantID,
		input.CustomerID,
		items,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("create order domain entity: %w", err)
	}

	kdsOrder, err := domain.NewKDSOrder(
		input.TenantID,
		order.ID,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("create kds entity: %w", err)
	}

	tx, err := uc.DB.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin order transaction: %w", err)
	}

	committed := false

	defer func() {
		if !committed {
			rollbackCtx := context.WithoutCancel(ctx)

			if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil &&
				!errors.Is(rollbackErr, context.Canceled) {
				// The original transaction error is more important.
				// Rollback failure is intentionally not returned here.
			}
		}
	}()

	if err := uc.Orders.Create(ctx, tx, order); err != nil {
		return nil, fmt.Errorf("persist order: %w", err)
	}

	if err := uc.KDS.Create(ctx, tx, kdsOrder); err != nil {
		return nil, fmt.Errorf("persist kds order: %w", err)
	}

	if err := uc.Inventory.DeductIngredientsForOrder(
		ctx,
		tx,
		input.TenantID,
		order,
	); err != nil {
		return nil, fmt.Errorf("deduct ingredients: %w", err)
	}

	event := domain.OrderCreatedEvent{
		EventID:     uuid.New(),
		TenantID:    input.TenantID,
		OrderID:     order.ID.UUID(),
		CustomerID:  input.CustomerID,
		State:       order.State,
		TotalMinor:  order.Total.MinorUnits,
		Currency:    order.Total.Currency,
		OccurredAtT: now,
	}

	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return nil, fmt.Errorf("append order created event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit order transaction: %w", err)
	}

	committed = true

	return order, nil
}
