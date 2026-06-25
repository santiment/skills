package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"santiment.net/san-skills/internal/hyperhandler/models"
)

func TestOrderResult(t *testing.T) {
	t.Run("U-ORD-01 successful", func(t *testing.T) {
		r := models.OrderResult{
			Success: true, OrderID: models.Ptr(int64(123)),
			FilledSize: dec("0.1"), AvgPrice: decP("67500"), Status: models.StatusFilled,
		}
		assert.True(t, r.Success)
		assert.Equal(t, int64(123), *r.OrderID)
		assert.True(t, r.IsFilled())
	})

	t.Run("U-ORD-02 failed with error", func(t *testing.T) {
		r := models.OrderResult{Success: false, Error: "Insufficient margin", Status: models.StatusRejected}
		assert.False(t, r.Success)
		assert.Equal(t, "Insufficient margin", r.Error)
	})

	t.Run("U-ORD-03 partial fill", func(t *testing.T) {
		r := models.OrderResult{
			Success: true, OrderID: models.Ptr(int64(456)),
			FilledSize: dec("0.05"), Status: models.StatusPartiallyFilled,
		}
		assert.True(t, r.IsPartial())
		assert.True(t, r.FilledSize.Equal(dec("0.05")))
	})
}

func TestPosition(t *testing.T) {
	t.Run("long position", func(t *testing.T) {
		p := models.Position{
			Coin: "BTC", Size: dec("0.1"), EntryPrice: dec("67500"),
			PositionValue: dec("6750"), UnrealizedPnl: dec("100"),
			Leverage: 5, LeverageType: "cross",
		}
		assert.True(t, p.IsLong())
		assert.False(t, p.IsShort())
		assert.True(t, p.AbsSize().Equal(dec("0.1")))
	})

	t.Run("short position", func(t *testing.T) {
		p := models.Position{
			Coin: "ETH", Size: dec("-1.0"), EntryPrice: dec("3500"),
			PositionValue: dec("3500"), UnrealizedPnl: dec("-50"),
			Leverage: 10, LeverageType: "isolated",
		}
		assert.False(t, p.IsLong())
		assert.True(t, p.IsShort())
		assert.True(t, p.AbsSize().Equal(dec("1.0")))
	})
}

func TestOpenOrder(t *testing.T) {
	t.Run("buy order", func(t *testing.T) {
		o := models.OpenOrder{Coin: "BTC", OrderID: 123, Side: "B", Price: dec("67000"), Size: dec("0.1"), Timestamp: 1699999999999}
		assert.True(t, o.IsBuy())
	})
	t.Run("sell order", func(t *testing.T) {
		o := models.OpenOrder{Coin: "BTC", OrderID: 456, Side: "S", Price: dec("68000"), Size: dec("0.1"), Timestamp: 1699999999999}
		assert.False(t, o.IsBuy())
	})
}
