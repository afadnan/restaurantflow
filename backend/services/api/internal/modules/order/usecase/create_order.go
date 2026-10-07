package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

type CreateOrderInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	TableID      *uuid.UUID
	CustomerID   *uuid.UUID

	OrderType     domain.OrderType
	PaymentStatus domain.PaymentStatus

	Currency       string
	TaxAmount      int64
	DiscountAmount int64
	ServiceFee     int64

	CustomerName  string
	CustomerPhone string
	Notes         string

	Items []CreateOrderItemInput
}

type CreateOrderItemInput struct {
	ProductID uuid.UUID
	Name      string
	Quantity  int32
	UnitPrice int64
	Notes     string
}

type CreateOrderUseCase struct {
	DB        *postgres.DB
	Orders    domain.OrderRepository
	KDS       domain.KDSRepository
	Inventory domain.InventoryRepository
	Events    domain.EventRepository
	Clock     func() time.Time
}

func (uc *CreateOrderUseCase) Execute(
	ctx context.Context,
	input CreateOrderInput,
) (*domain.Order, error) {
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if input.RestaurantID == uuid.Nil {
		return nil, domain.ErrInvalidRestaurantID
	}

	now := time.Now().UTC()

	if uc.Clock != nil {
		now = uc.Clock().UTC()
	}

	currency := input.Currency

	if currency == "" {
		currency = "INR"
	}

	items := make([]domain.OrderItem, 0, len(input.Items))

	for _, itemInput := range input.Items {
		money, err := domain.NewMoney(
			itemInput.UnitPrice,
			currency,
		)
		if err != nil {
			return nil, err
		}

		item, err := domain.NewOrderItem(
			itemInput.ProductID,
			itemInput.Name,
			itemInput.Quantity,
			money,
			itemInput.Notes,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	order, err := domain.NewOrder(domain.NewOrderParams{
		TenantID:       input.TenantID,
		RestaurantID:   input.RestaurantID,
		TableID:        input.TableID,
		CustomerID:     input.CustomerID,
		OrderType:      input.OrderType,
		PaymentStatus:  input.PaymentStatus,
		Currency:       currency,
		TaxAmount:      input.TaxAmount,
		DiscountAmount: input.DiscountAmount,
		ServiceFee:     input.ServiceFee,
		CustomerName:   input.CustomerName,
		CustomerPhone:  input.CustomerPhone,
		Notes:          input.Notes,
		Items:          items,
		Now:            now,
	})
	if err != nil {
		return nil, err
	}

	kdsOrder, err := domain.NewKDSOrder(
		order.TenantID.UUID(),
		order.RestaurantID.UUID(),
		order.ID,
		now,
	)
	if err != nil {
		return nil, err
	}

	tx, err := uc.DB.BeginTx(ctx, order.TenantID.UUID())
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := uc.Orders.Create(ctx, tx, order); err != nil {
		return nil, err
	}

	if err := uc.KDS.Create(ctx, tx, kdsOrder); err != nil {
		return nil, err
	}

	if err := uc.Inventory.DeductIngredientsForOrder(
		ctx,
		tx,
		order.TenantID.UUID(),
		order,
	); err != nil {
		return nil, err
	}

	var customerID uuid.UUID

	if order.CustomerID != nil {
		customerID = order.CustomerID.UUID()
	}

	event := domain.OrderCreatedEvent{
		ID:         uuid.New(),
		Tenant:     order.TenantID.UUID(),
		OrderID:    order.ID.UUID(),
		CustomerID: customerID,
		State:      order.State,
		TotalMinor: order.Total.MinorUnits,
		Currency:   order.Total.Currency,
		Occurred:   now,
	}

	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return order, nil
}
