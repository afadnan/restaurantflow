package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
)

var (
	errTenantIDRequired = errors.New("tenant id is required")
	errOrderRequired    = errors.New("order is required")
	errOrderIDRequired  = errors.New("order id is required")
)

type PostgresInventoryRepository struct{}

func NewPostgresInventoryRepository() *PostgresInventoryRepository {
	return &PostgresInventoryRepository{}
}

func (r *PostgresInventoryRepository) DeductIngredientsForOrder(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	order *domain.Order,
) error {
	if tenantID == uuid.Nil {
		return errTenantIDRequired
	}

	if order == nil {
		return errOrderRequired
	}

	orderID := order.ID.UUID()
	if orderID == uuid.Nil {
		return errOrderIDRequired
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	q := db.New(pgTx.Raw())

	rows, err := q.CheckInventoryForOrder(
		ctx,
		db.CheckInventoryForOrderParams{
			TenantID: uuidToPgUUID(tenantID),
			OrderID:  uuidToPgUUID(orderID),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"check inventory for order %s: %w",
			orderID,
			err,
		)
	}

	// An order without ingredient requirements does not need
	// an inventory mutation.
	if len(rows) == 0 {
		return nil
	}

	for _, row := range rows {
		required, err := numericToThousandths(row.RequiredQuantity)
		if err != nil {
			return fmt.Errorf(
				"invalid required inventory quantity for ingredient %s: %w",
				row.IngredientID,
				err,
			)
		}

		available, err := numericToThousandths(row.AvailableQuantity)
		if err != nil {
			return fmt.Errorf(
				"invalid available inventory quantity for ingredient %s: %w",
				row.IngredientID,
				err,
			)
		}

		if available < required {
			return fmt.Errorf(
				"insufficient inventory for ingredient %s: required %s, available %s",
				row.IngredientID,
				formatThousandths(required),
				formatThousandths(available),
			)
		}
	}

	updatedRows, err := q.DeductInventoryForOrder(
		ctx,
		db.DeductInventoryForOrderParams{
			TenantID: uuidToPgUUID(tenantID),
			OrderID:  uuidToPgUUID(orderID),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"deduct inventory for order %s: %w",
			orderID,
			err,
		)
	}

	if updatedRows != int64(len(rows)) {
		return fmt.Errorf(
			"inventory deduction affected %d rows, expected %d",
			updatedRows,
			len(rows),
		)
	}

	return nil
}

// numericToThousandths converts a PostgreSQL numeric quantity into
// an integer representation with three decimal places.
//
// Example:
//
//	1      -> 1000
//	1.25   -> 1250
//	0.005  -> 5
//
// PostgreSQL numeric values are returned by pgx as a string-compatible
// value, so this function deliberately parses the textual representation
// rather than going through float64.
func numericToThousandths(value any) (int64, error) {
	s := strings.TrimSpace(fmt.Sprint(value))
	if s == "" {
		return 0, errors.New("empty numeric quantity")
	}

	negative := false

	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}

	if s == "" {
		return 0, errors.New("invalid numeric quantity")
	}

	parts := strings.SplitN(s, ".", 2)

	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid whole quantity %q: %w", value, err)
	}

	var fractional int64

	if len(parts) == 2 {
		fraction := parts[1]

		if len(fraction) > 3 {
			// Inventory quantities are represented at thousandth
			// precision. Reject values that cannot be represented
			// exactly instead of silently rounding.
			if strings.TrimRight(fraction[3:], "0") != "" {
				return 0, fmt.Errorf(
					"quantity %q has more than three decimal places",
					value,
				)
			}

			fraction = fraction[:3]
		}

		for len(fraction) < 3 {
			fraction += "0"
		}

		if fraction != "" {
			fractional, err = strconv.ParseInt(fraction, 10, 64)
			if err != nil {
				return 0, fmt.Errorf(
					"invalid fractional quantity %q: %w",
					value,
					err,
				)
			}
		}
	}

	if whole > math.MaxInt64/1000 {
		return 0, fmt.Errorf("quantity %q overflows int64", value)
	}

	result := whole * 1000

	if result > math.MaxInt64-fractional {
		return 0, fmt.Errorf("quantity %q overflows int64", value)
	}

	result += fractional

	if negative {
		return -result, nil
	}

	return result, nil
}

func formatThousandths(value int64) string {
	negative := value < 0

	if negative {
		value = -value
	}

	whole := value / 1000
	fraction := value % 1000

	result := fmt.Sprintf("%d.%03d", whole, fraction)

	if negative {
		return "-" + result
	}

	return result
}

var _ domain.InventoryRepository = (*PostgresInventoryRepository)(nil)
