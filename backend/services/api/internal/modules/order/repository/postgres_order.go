package repository

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
)

type PostgresOrderRepository struct{}

func NewPostgresOrderRepository() *PostgresOrderRepository {
	return &PostgresOrderRepository{}
}

func (r *PostgresOrderRepository) Create(
	ctx context.Context,
	tx domain.Transaction,
	order *domain.Order,
) error {
	if order == nil {
		return errors.New("order is required")
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	q := db.New(pgTx.Raw())

	orderRow, err := q.CreateOrder(ctx, db.CreateOrderParams{
		TenantID:       uuidToPgUUID(order.TenantID.UUID()),
		RestaurantID:   uuidToPgUUID(order.RestaurantID.UUID()),
		TableID:        optionalUUIDToPgUUID(order.TableID),
		CustomerID:     optionalUUIDToPgUUID(order.CustomerID),
		OrderType:      db.OrderType(order.Type),
		Status:         db.OrderStatus(order.State),
		PaymentStatus:  db.PaymentStatus(order.PaymentStatus),
		Subtotal:       moneyToNumeric(order.Subtotal),
		TaxAmount:      moneyToNumeric(order.TaxAmount),
		DiscountAmount: moneyToNumeric(order.DiscountAmount),
		ServiceFee:     moneyToNumeric(order.ServiceFee),
		TotalAmount:    moneyToNumeric(order.Total),
		Currency:       order.Currency,
		CustomerName:   textToPgText(order.CustomerName),
		CustomerPhone:  textToPgText(order.CustomerPhone),
		Notes:          textToPgText(order.Notes),
	})
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	// PostgreSQL generates the order UUID when the INSERT does not
	// explicitly provide one. Reconstruct the domain OrderID through
	// its constructor so domain invariants remain enforced.
	orderUUID, err := pgUUIDToUUID(orderRow.ID)
	if err != nil {
		return fmt.Errorf("created order id: %w", err)
	}

	orderID, err := domain.NewOrderIDFromUUID(orderUUID)
	if err != nil {
		return fmt.Errorf("created order id: %w", err)
	}

	order.ID = orderID

	if orderRow.OrderNumber.Valid {
		order.OrderNumber = orderRow.OrderNumber.Int64
	}

	if orderRow.CreatedAt.Valid {
		order.CreatedAt = orderRow.CreatedAt.Time
	}

	if orderRow.UpdatedAt.Valid {
		order.UpdatedAt = orderRow.UpdatedAt.Time
	}

	for _, item := range order.Items {
		_, err := q.CreateOrderItem(ctx, db.CreateOrderItemParams{
			TenantID:   uuidToPgUUID(order.TenantID.UUID()),
			OrderID:    uuidToPgUUID(order.ID.UUID()),
			MenuItemID: uuidToPgUUID(item.ProductID.UUID()),
			ItemName:   item.Name,
			UnitPrice:  moneyToNumeric(item.UnitPrice),
			Quantity:   item.Quantity.Value(),
			Subtotal:   moneyToNumeric(item.Subtotal),
			Notes:      textToPgText(item.Notes),
		})
		if err != nil {
			return fmt.Errorf(
				"create order item for order %s: %w",
				order.ID.UUID(),
				err,
			)
		}
	}

	return nil
}

func (r *PostgresOrderRepository) GetByID(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	orderID uuid.UUID,
) (*domain.Order, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant id is required")
	}

	if orderID == uuid.Nil {
		return nil, errors.New("order id is required")
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	q := db.New(pgTx.Raw())

	orderRow, err := q.GetOrderByID(ctx, db.GetOrderByIDParams{
		TenantID: uuidToPgUUID(tenantID),
		ID:       uuidToPgUUID(orderID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}

		return nil, fmt.Errorf("get order %s: %w", orderID, err)
	}

	itemRows, err := q.ListOrderItems(ctx, db.ListOrderItemsParams{
		TenantID: uuidToPgUUID(tenantID),
		OrderID:  uuidToPgUUID(orderID),
	})
	if err != nil {
		return nil, fmt.Errorf(
			"list order items for %s: %w",
			orderID,
			err,
		)
	}

	order, err := mapOrder(orderRow, itemRows)
	if err != nil {
		return nil, fmt.Errorf("map order %s: %w", orderID, err)
	}

	return order, nil
}

func (r *PostgresOrderRepository) UpdateState(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	orderID uuid.UUID,
	from domain.OrderState,
	to domain.OrderState,
) error {
	if tenantID == uuid.Nil {
		return errors.New("tenant id is required")
	}

	if orderID == uuid.Nil {
		return errors.New("order id is required")
	}

	if !from.Valid() {
		return fmt.Errorf("invalid current order state: %s", from)
	}

	if !to.Valid() {
		return fmt.Errorf("invalid target order state: %s", to)
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	q := db.New(pgTx.Raw())

	current, err := q.GetOrderByID(ctx, db.GetOrderByIDParams{
		TenantID: uuidToPgUUID(tenantID),
		ID:       uuidToPgUUID(orderID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrOrderNotFound
		}

		return fmt.Errorf("get order before state update: %w", err)
	}

	currentState := domain.OrderState(current.Status)

	if currentState != from {
		return fmt.Errorf(
			"order %s state conflict: expected %s, actual %s",
			orderID,
			from,
			currentState,
		)
	}

	if err := validateOrderTransition(from, to); err != nil {
		return err
	}

	updated, err := q.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		TenantID: uuidToPgUUID(tenantID),
		ID:       uuidToPgUUID(orderID),
		Status:   db.OrderStatus(to),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrOrderNotFound
		}

		return fmt.Errorf(
			"update order %s state: %w",
			orderID,
			err,
		)
	}

	if domain.OrderState(updated.Status) != to {
		return fmt.Errorf(
			"order %s state update returned unexpected state %s",
			orderID,
			updated.Status,
		)
	}

	return nil
}

func validateOrderTransition(
	from domain.OrderState,
	to domain.OrderState,
) error {
	switch from {
	case domain.OrderStatePending:
		switch to {
		case domain.OrderStateConfirmed,
			domain.OrderStatePreparing,
			domain.OrderStateCancelled:
			return nil
		}

	case domain.OrderStateConfirmed:
		switch to {
		case domain.OrderStatePreparing,
			domain.OrderStateCancelled:
			return nil
		}

	case domain.OrderStatePreparing:
		switch to {
		case domain.OrderStateReady,
			domain.OrderStateCancelled:
			return nil
		}

	case domain.OrderStateReady:
		switch to {
		case domain.OrderStateCompleted,
			domain.OrderStateCancelled:
			return nil
		}

	case domain.OrderStateCompleted,
		domain.OrderStateCancelled:
		return domain.ErrInvalidStateTransition
	}

	return domain.ErrInvalidStateTransition
}

func mapOrder(
	row db.Order,
	itemRows []db.OrderItem,
) (*domain.Order, error) {
	tenantUUID, err := pgUUIDToUUID(row.TenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant id: %w", err)
	}

	tenantID, err := domain.NewTenantID(tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("tenant id: %w", err)
	}

	restaurantUUID, err := pgUUIDToUUID(row.RestaurantID)
	if err != nil {
		return nil, fmt.Errorf("restaurant id: %w", err)
	}

	restaurantID, err := domain.NewRestaurantID(restaurantUUID)
	if err != nil {
		return nil, fmt.Errorf("restaurant id: %w", err)
	}

	orderUUID, err := pgUUIDToUUID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("order id: %w", err)
	}

	orderID, err := domain.NewOrderIDFromUUID(orderUUID)
	if err != nil {
		return nil, fmt.Errorf("order id: %w", err)
	}

	orderType := domain.OrderType(row.OrderType)
	if !orderType.Valid() {
		return nil, fmt.Errorf(
			"invalid persisted order type %q",
			row.OrderType,
		)
	}

	state := domain.OrderState(row.Status)
	if !state.Valid() {
		return nil, fmt.Errorf(
			"invalid persisted order state %q",
			row.Status,
		)
	}

	paymentStatus := domain.PaymentStatus(row.PaymentStatus)
	if !paymentStatus.Valid() {
		return nil, fmt.Errorf(
			"invalid persisted payment status %q",
			row.PaymentStatus,
		)
	}

	items := make([]domain.OrderItem, 0, len(itemRows))

	for _, itemRow := range itemRows {
		productUUID, err := pgUUIDToUUID(itemRow.MenuItemID)
		if err != nil {
			return nil, fmt.Errorf(
				"order item product id: %w",
				err,
			)
		}

		productID, err := domain.NewProductID(productUUID)
		if err != nil {
			return nil, fmt.Errorf(
				"order item product id: %w",
				err,
			)
		}

		itemUUID, err := pgUUIDToUUID(itemRow.ID)
		if err != nil {
			return nil, fmt.Errorf(
				"order item id: %w",
				err,
			)
		}

		unitPrice, err := numericToMoney(
			itemRow.UnitPrice,
			row.Currency,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"order item unit price: %w",
				err,
			)
		}

		subtotal, err := numericToMoney(
			itemRow.Subtotal,
			row.Currency,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"order item subtotal: %w",
				err,
			)
		}

		quantity, err := domain.NewQuantity(itemRow.Quantity)
		if err != nil {
			return nil, fmt.Errorf(
				"order item quantity: %w",
				err,
			)
		}

		items = append(items, domain.OrderItem{
			ID:        itemUUID,
			ProductID: productID,
			Name:      itemRow.ItemName,
			Quantity:  quantity,
			UnitPrice: unitPrice,
			Subtotal:  subtotal,
			Notes:     pgTextToString(itemRow.Notes),
		})
	}

	subtotal, err := numericToMoney(
		row.Subtotal,
		row.Currency,
	)
	if err != nil {
		return nil, fmt.Errorf("subtotal: %w", err)
	}

	taxAmount, err := numericToMoney(
		row.TaxAmount,
		row.Currency,
	)
	if err != nil {
		return nil, fmt.Errorf("tax amount: %w", err)
	}

	discountAmount, err := numericToMoney(
		row.DiscountAmount,
		row.Currency,
	)
	if err != nil {
		return nil, fmt.Errorf("discount amount: %w", err)
	}

	serviceFee, err := numericToMoney(
		row.ServiceFee,
		row.Currency,
	)
	if err != nil {
		return nil, fmt.Errorf("service fee: %w", err)
	}

	total, err := numericToMoney(
		row.TotalAmount,
		row.Currency,
	)
	if err != nil {
		return nil, fmt.Errorf("total amount: %w", err)
	}

	order := &domain.Order{
		ID:             orderID,
		TenantID:       tenantID,
		RestaurantID:   restaurantID,
		Type:           orderType,
		State:          state,
		PaymentStatus:  paymentStatus,
		Items:          items,
		Subtotal:       subtotal,
		TaxAmount:      taxAmount,
		DiscountAmount: discountAmount,
		ServiceFee:     serviceFee,
		Total:          total,
		Currency:       row.Currency,
		CustomerName:   pgTextToString(row.CustomerName),
		CustomerPhone:  pgTextToString(row.CustomerPhone),
		Notes:          pgTextToString(row.Notes),
		OrderNumber:    row.OrderNumber.Int64,
	}

	if row.TableID.Valid {
		tableUUID, err := pgUUIDToUUID(row.TableID)
		if err != nil {
			return nil, fmt.Errorf("table id: %w", err)
		}

		tableID, err := domain.NewTableID(tableUUID)
		if err != nil {
			return nil, fmt.Errorf("table id: %w", err)
		}

		order.TableID = &tableID
	}

	if row.CustomerID.Valid {
		customerUUID, err := pgUUIDToUUID(row.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("customer id: %w", err)
		}

		customerID, err := domain.NewCustomerID(customerUUID)
		if err != nil {
			return nil, fmt.Errorf("customer id: %w", err)
		}

		order.CustomerID = &customerID
	}

	if row.CreatedAt.Valid {
		order.CreatedAt = row.CreatedAt.Time
	}

	if row.UpdatedAt.Valid {
		order.UpdatedAt = row.UpdatedAt.Time
	}

	if row.CompletedAt.Valid {
		t := row.CompletedAt.Time
		order.CompletedAt = &t
	}

	if row.CancelledAt.Valid {
		t := row.CancelledAt.Time
		order.CancelledAt = &t
	}

	return order, nil
}

func postgresTx(tx domain.Transaction) (*postgres.Tx, error) {
	pgTx, ok := tx.(*postgres.Tx)
	if !ok || pgTx == nil {
		return nil, errors.New(
			"transaction is not a postgres transaction",
		)
	}

	return pgTx, nil
}

func uuidToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: id != uuid.Nil,
	}
}

func optionalUUIDToPgUUID[T interface {
	UUID() uuid.UUID
}](id *T) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}

	return uuidToPgUUID((*id).UUID())
}

func moneyToNumeric(m domain.Money) pgtype.Numeric {
	return pgtype.Numeric{
		Int:   big.NewInt(m.MinorUnits),
		Exp:   -2,
		Valid: true,
	}
}

func numericToMoney(
	n pgtype.Numeric,
	currency string,
) (domain.Money, error) {
	if !n.Valid {
		return domain.Money{}, domain.ErrInvalidPrice
	}

	if n.Int == nil {
		return domain.Money{}, domain.ErrInvalidPrice
	}

	// The domain stores money as integer minor units.
	//
	// Example:
	//
	// 12550 minor units
	//       ↓
	// NUMERIC Int = 12550
	// NUMERIC Exp = -2
	//       ↓
	// 125.50
	//
	// Therefore the Int value can be reconstructed directly
	// as the domain's minor-unit amount.
	minorUnits := n.Int.Int64()

	return domain.NewMoney(minorUnits, currency)
}

func pgUUIDToUUID(value pgtype.UUID) (uuid.UUID, error) {
	if !value.Valid {
		return uuid.Nil, errors.New("uuid is NULL")
	}

	return value.Bytes, nil
}

func textToPgText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: value,
		Valid:  true,
	}
}

func pgTextToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

var _ domain.OrderRepository = (*PostgresOrderRepository)(nil)
