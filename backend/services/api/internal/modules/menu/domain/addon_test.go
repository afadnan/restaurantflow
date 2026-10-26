package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewAddon(t *testing.T) {
	tenantID := uuid.New()
	restaurantID := uuid.New()
	now := time.Now().UTC()

	addon, err := NewAddon(
		tenantID,
		restaurantID,
		" Extra Cheese ",
		" Mozzarella ",
		"25.50",
		now,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addon.Name != "Extra Cheese" {
		t.Fatalf("unexpected name: %q", addon.Name)
	}
	if addon.Description != "Mozzarella" {
		t.Fatalf("unexpected description: %q", addon.Description)
	}
	if addon.Price != "25.50" || !addon.IsActive {
		t.Fatalf("unexpected addon state: %+v", addon)
	}
}

func TestNewAddonValidation(t *testing.T) {
	tenantID := uuid.New()
	restaurantID := uuid.New()

	tests := []struct {
		name         string
		tenantID     uuid.UUID
		restaurantID uuid.UUID
		addonName    string
		price        string
		wantErr      error
	}{
		{"missing tenant", uuid.Nil, restaurantID, "Cheese", "10", ErrInvalidTenantID},
		{"missing restaurant", tenantID, uuid.Nil, "Cheese", "10", ErrInvalidRestaurantID},
		{"empty name", tenantID, restaurantID, " ", "10", ErrInvalidAddonName},
		{"negative price", tenantID, restaurantID, "Cheese", "-1", ErrInvalidAddonPrice},
		{"excess precision", tenantID, restaurantID, "Cheese", "10.999", ErrInvalidAddonPrice},
		{"invalid price", tenantID, restaurantID, "Cheese", "abc", ErrInvalidAddonPrice},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAddon(
				tt.tenantID, tt.restaurantID,
				tt.addonName, "", tt.price, time.Now(),
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestAddonUpdateAndDeactivate(t *testing.T) {
	tenantID := uuid.New()
	restaurantID := uuid.New()
	now := time.Now().UTC()

	addon, err := NewAddon(
		tenantID, restaurantID, "Cheese", "", "20", now,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = addon.Update(
		tenantID, restaurantID, addon.ID,
		"Extra Cheese", "Premium", "30.00",
		now.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if addon.Name != "Extra Cheese" || addon.Price != "30.00" {
		t.Fatalf("update failed: %+v", addon)
	}

	err = addon.Deactivate(
		tenantID, restaurantID, addon.ID,
		now.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if addon.IsActive {
		t.Fatal("addon should be inactive")
	}

	err = addon.Update(
		tenantID, restaurantID, addon.ID,
		"Another", "", "40", now.Add(3*time.Minute),
	)
	if !errors.Is(err, ErrAddonInactive) {
		t.Fatalf("expected ErrAddonInactive, got %v", err)
	}
}

func TestNewItemAddon(t *testing.T) {
	tenantID := uuid.New()
	restaurantID := uuid.New()
	itemID := uuid.New()
	addonID := uuid.New()

	association, err := NewItemAddon(
		tenantID, restaurantID, itemID, addonID, true, 2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !association.IsRequired || association.MaxQuantity != 2 {
		t.Fatalf("unexpected association: %+v", association)
	}

	err = association.Update(
		tenantID, restaurantID, itemID, addonID, false, 3,
	)
	if err != nil {
		t.Fatal(err)
	}
	if association.IsRequired || association.MaxQuantity != 3 {
		t.Fatalf("association update failed: %+v", association)
	}

	_, err = NewItemAddon(
		tenantID, restaurantID, itemID, addonID, false, 0,
	)
	if !errors.Is(err, ErrInvalidAddonMaxQuantity) {
		t.Fatalf("expected invalid quantity, got %v", err)
	}

	err = association.Update(
		tenantID, uuid.New(), itemID, addonID, false, 4,
	)
	if !errors.Is(err, ErrInvalidRestaurantID) {
		t.Fatalf("expected restaurant ownership error, got %v", err)
	}
}
