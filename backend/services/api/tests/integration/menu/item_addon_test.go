package menu_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
)

func TestIntegrationMenuItemAddonRepository(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addonRepo := repository.NewPostgresAddonRepository()
	linkRepo := repository.NewPostgresItemAddonRepository()

	addon, err := domain.NewAddon(
		fixture.tenantID,
		fixture.restaurantID,
		"Extra Sauce",
		"Optional sauce",
		"15.00",
		time.Now().UTC(),
	)
	require.NoError(t, err)

	tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)
	require.NoError(t, addonRepo.Create(ctx, tx, addon))
	require.NoError(t, tx.Commit(ctx))

	newLink := func() *domain.ItemAddon {
		t.Helper()
		link, err := domain.NewItemAddon(
			fixture.tenantID,
			fixture.restaurantID,
			fixture.itemID,
			addon.ID,
			false,
			2,
		)
		require.NoError(t, err)
		return link
	}

	t.Run("attach and commit", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		linked, err := linkRepo.Attach(ctx, tx, newLink())
		require.NoError(t, err)
		require.Equal(t, addon.ID, linked.AddonID)
		require.Equal(t, int32(2), linked.MaxQuantity)
		require.NoError(t, tx.Commit(ctx))
	})

	t.Run("get and list", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		linked, err := linkRepo.GetByID(
			ctx, tx,
			fixture.tenantID, fixture.restaurantID,
			fixture.itemID, addon.ID,
		)
		require.NoError(t, err)
		require.Equal(t, fixture.itemID, linked.ItemID)

		links, err := linkRepo.List(
			ctx, tx,
			fixture.tenantID, fixture.restaurantID,
			fixture.itemID, 50, 0,
		)
		require.NoError(t, err)
		require.Len(t, links, 1)
	})

	t.Run("duplicate attachment rejected", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = linkRepo.Attach(ctx, tx, newLink())
		require.ErrorIs(t, err, domain.ErrItemAddonAlreadyExists)
	})

	t.Run("invalid quantity rejected", func(t *testing.T) {
		link := newLink()
		link.MaxQuantity = 0

		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = linkRepo.Update(ctx, tx, link)
		require.ErrorIs(t, err, domain.ErrInvalidAddonMaxQuantity)
	})

	t.Run("wrong restaurant cannot read or update", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = linkRepo.GetByID(
			ctx, tx,
			fixture.tenantID, uuid.New(),
			fixture.itemID, addon.ID,
		)
		require.ErrorIs(t, err, domain.ErrItemAddonNotFound)

		link := newLink()
		link.RestaurantID = uuid.New()
		_, err = linkRepo.Update(ctx, tx, link)
		require.ErrorIs(t, err, domain.ErrItemAddonNotFound)
	})

	t.Run("update association", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		link := newLink()
		link.IsRequired = true
		link.MaxQuantity = 3

		updated, err := linkRepo.Update(ctx, tx, link)
		require.NoError(t, err)
		require.True(t, updated.IsRequired)
		require.Equal(t, int32(3), updated.MaxQuantity)
		require.NoError(t, tx.Commit(ctx))
	})

	t.Run("cross restaurant attachment rejected", func(t *testing.T) {
		otherRestaurantID := uuid.New()

		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)

		_, err = tx.Raw().Exec(ctx, `
			INSERT INTO restaurants (id, tenant_id, name, slug)
			VALUES ($1, $2, $3, $4)
		`, otherRestaurantID, fixture.tenantID,
			"Other Addon Restaurant",
			"other-addon-"+otherRestaurantID.String())
		require.NoError(t, err)
		require.NoError(t, tx.Commit(ctx))

		foreignAddon, err := domain.NewAddon(
			fixture.tenantID,
			otherRestaurantID,
			"Other Restaurant Sauce",
			"",
			"10.00",
			time.Now().UTC(),
		)
		require.NoError(t, err)

		tx, err = integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		require.NoError(t, addonRepo.Create(ctx, tx, foreignAddon))
		require.NoError(t, tx.Commit(ctx))

		foreignLink, err := domain.NewItemAddon(
			fixture.tenantID,
			fixture.restaurantID,
			fixture.itemID,
			foreignAddon.ID,
			false,
			1,
		)
		require.NoError(t, err)

		tx, err = integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = linkRepo.Attach(ctx, tx, foreignLink)
		require.ErrorIs(t, err, domain.ErrInvalidItemAddonInput)
	})

	t.Run("detach and reject missing association", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = linkRepo.Detach(
			ctx, tx,
			fixture.tenantID, fixture.restaurantID,
			fixture.itemID, addon.ID,
		)
		require.NoError(t, err)
		require.NoError(t, tx.Commit(ctx))

		verifyTx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = verifyTx.Rollback(ctx) }()

		_, err = linkRepo.GetByID(
			ctx, verifyTx,
			fixture.tenantID, fixture.restaurantID,
			fixture.itemID, addon.ID,
		)
		require.ErrorIs(t, err, domain.ErrItemAddonNotFound)
	})

	t.Run("inactive addon cannot be attached", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		_, err = addonRepo.Deactivate(
			ctx, tx,
			fixture.tenantID, fixture.restaurantID,
			addon.ID, time.Now().UTC(),
		)
		require.NoError(t, err)
		require.NoError(t, tx.Commit(ctx))

		tx, err = integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = linkRepo.Attach(ctx, tx, newLink())
		require.ErrorIs(t, err, domain.ErrInvalidItemAddonInput)
	})
}
