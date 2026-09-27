package repository

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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

// numericToThousandths converts a PostgreSQL NUMERIC quantity into
// an exact integer representation with three decimal places.
//
// PostgreSQL inventory quantities use NUMERIC(14,3), so:
//
//	1      -> 1000
//	1.25   -> 1250
//	0.005  -> 5
//
// The conversion is performed using math/big rather than float64 so
// that inventory quantities remain exact.
func numericToThousandths(value pgtype.Numeric) (int64, error) {
	if !value.Valid {
		return 0, errors.New("invalid numeric quantity")
	}

	if value.NaN {
		return 0, errors.New("numeric quantity is NaN")
	}

	if value.InfinityModifier != pgtype.Finite {
		return 0, errors.New("numeric quantity is infinite")
	}

	if value.Int == nil {
		return 0, errors.New("numeric quantity has no integer value")
	}

	// PostgreSQL NUMERIC represents:
	//
	//     value = Int * 10^Exp
	//
	// We need:
	//
	//     thousandths = value * 1000
	//
	// Therefore:
	//
	//     thousandths = Int * 10^(Exp + 3)
	//
	result := new(big.Int).Set(value.Int)
	scale := int(value.Exp) + 3

	if scale > 0 {
		multiplier := new(big.Int).Exp(
			big.NewInt(10),
			big.NewInt(int64(scale)),
			nil,
		)
		result.Mul(result, multiplier)
	} else if scale < 0 {
		divisor := new(big.Int).Exp(
			big.NewInt(10),
			big.NewInt(int64(-scale)),
			nil,
		)

		quotient := new(big.Int)
		remainder := new(big.Int)

		quotient.QuoRem(result, divisor, remainder)

		// Inventory quantities must be exactly representable at
		// thousandth precision. Do not silently round.
		if remainder.Sign() != 0 {
			return 0, fmt.Errorf(
				"numeric quantity %v has more than three decimal places",
				value,
			)
		}

		result = quotient
	}

	if !result.IsInt64() {
		return 0, fmt.Errorf(
			"numeric quantity %v overflows int64",
			value,
		)
	}

	return result.Int64(), nil
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
