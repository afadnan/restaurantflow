package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewVariant(t *testing.T) {
	tenantID := uuid.New()
	restaurantID := uuid.New()
	itemID := uuid.New()
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.FixedZone("IST", 5*60*60+30*60))

	tests := []struct {
		name       string
		variant    string
		priceDelta string
		wantErr    error
	}{
		{
			name:       "valid positive delta",
			variant:    " Large ",
			priceDelta: "25.50",
		},
		{
			name:       "valid negative delta",
			variant:    "Small",
			priceDelta: "-20.00",
		},
		{
			name:       "valid zero delta",
			variant:    "Regular",
			priceDelta: "0",
		},
		{
			name:       "empty name",
			variant:    "  ",
			priceDelta: "0",
			wantErr:    ErrInvalidVariantName,
		},
		{
			name:       "too many fractional digits",
			variant:    "Large",
			priceDelta: "1.234",
			wantErr:    ErrInvalidPriceDelta,
		},
		{
			name:       "too many integer digits",
			variant:    "Large",
			priceDelta: "12345678901",
			wantErr:    ErrInvalidPriceDelta,
		},
		{
			name:       "scientific notation",
			variant:    "Large",
			priceDelta: "1e2",
			wantErr:    ErrInvalidPriceDelta,
		},
		{
			name:       "plus sign rejected",
			variant:    "Large",
			priceDelta: "+10",
			wantErr:    ErrInvalidPriceDelta,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewVariant(
				tenantID,
				restaurantID,
				itemID,
				tt.variant,
				tt.priceDelta,
				now,
			)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewVariant returned error: %v", err)
			}
			if got.ID == uuid.Nil {
				t.Fatal("expected generated variant ID")
			}
			if got.Name == "" || got.Name != "Large" && tt.name == "valid positive delta" {
				t.Fatalf("unexpected normalized name: %q", got.Name)
			}
			if !got.IsActive {
				t.Fatal("new variant should be active")
			}
			if !got.CreatedAt.Equal(now.UTC()) || !got.UpdatedAt.Equal(now.UTC()) {
				t.Fatal("timestamps should be normalized to UTC")
			}
		})
	}
}

func TestNewVariantRejectsInvalidIDs(t *testing.T) {
	now := time.Now()
	restaurantID := uuid.New()
	itemID := uuid.New()

	if _, err := NewVariant(uuid.Nil, restaurantID, itemID, "Regular", "0", now); !errors.Is(err, ErrInvalidTenantID) {
		t.Fatalf("expected invalid tenant error, got %v", err)
	}

	if _, err := NewVariant(uuid.New(), uuid.Nil, itemID, "Regular", "0", now); !errors.Is(err, ErrInvalidRestaurantID) {
		t.Fatalf("expected invalid restaurant error, got %v", err)
	}

	if _, err := NewVariant(uuid.New(), restaurantID, uuid.Nil, "Regular", "0", now); !errors.Is(err, ErrInvalidItemID) {
		t.Fatalf("expected invalid item error, got %v", err)
	}
}

func TestVariantUpdateAndDeactivate(t *testing.T) {
	tenantID := uuid.New()
	restaurantID := uuid.New()
	itemID := uuid.New()
	now := time.Now().UTC()

	variant, err := NewVariant(tenantID, restaurantID, itemID, "Regular", "0", now)
	if err != nil {
		t.Fatal(err)
	}

	updatedAt := now.Add(time.Minute)
	err = variant.Update(tenantID, restaurantID, itemID, variant.ID, "Large", "-15.50", updatedAt)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if variant.Name != "Large" || variant.PriceDelta != "-15.50" {
		t.Fatalf("unexpected updated variant: %+v", variant)
	}
	if !variant.UpdatedAt.Equal(updatedAt) {
		t.Fatal("updated timestamp was not applied")
	}

	deactivatedAt := updatedAt.Add(time.Minute)
	if err := variant.Deactivate(tenantID, restaurantID, itemID, variant.ID, deactivatedAt); err != nil {
		t.Fatalf("Deactivate returned error: %v", err)
	}
	if variant.IsActive {
		t.Fatal("variant should be inactive")
	}
	if !variant.UpdatedAt.Equal(deactivatedAt) {
		t.Fatal("deactivation timestamp was not applied")
	}

	if err := variant.Update(tenantID, restaurantID, itemID, variant.ID, "Small", "0", deactivatedAt); !errors.Is(err, ErrVariantInactive) {
		t.Fatalf("expected inactive error, got %v", err)
	}
}
