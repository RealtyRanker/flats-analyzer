// Package scoring mirrors realty-parser's scoring formula (kept in sync
// manually — see that package for the authoritative rent/sale constants).
// It's duplicated here so a subscriber's custom scoring weights (stored in
// subscription_scoring_params) can be applied at match time: realty-parser
// only ever computes one default FlatScore per flat, but each subscriber may
// want a different weighting, so flats-analyzer needs its own copy of the
// formula to recompute a personalized score on demand.
package scoring

import (
	"math"

	"github.com/asmisnik/flats-analyzer/internal/model"
)

type scoreConstants struct {
	defaultScore             float64
	depositMultiplier        float64
	allAreaMultiplier        float64
	costMultiplier           float64
	maxPossiblePrice         float64
	livingAreaMultiplier     float64
	livingAreaPart           float64
	kitchenAreaMultiplier    float64
	maxKitchenArea           float64
	defaultKitchenArea       float64
	comissionMultiplier      float64
	prepaidMonthsMultiplier  float64
	petsMultiplier           float64
	dishwasherMultiplier     float64
	conditionerMultiplier    float64
	apartmentsMultiplier     float64
	twoRoomMultiplier        float64
	threeRoomMultiplier      float64
	fourRoomMultiplier       float64
	additionalRoomsMultipier float64
	windowsYardMultiplier    float64
	windowsStreetMultiplier  float64
	windowsBothMultiplier    float64
	renovationCosmeticMult   float64
	renovationEuroMult       float64
	renovationDesignMult     float64
	bathroomSeparatedMult    float64
	balconyMultiplier        float64
	loggiaMultiplier         float64
	defaultUndergroundScore  float64
	undergroundScoreMult     float64
}

var rentConstants = scoreConstants{
	defaultScore:             600000.0,
	depositMultiplier:        0.02,
	allAreaMultiplier:        300.0,
	costMultiplier:           1.0,
	maxPossiblePrice:         300000,
	livingAreaMultiplier:     600.0,
	livingAreaPart:           0.5,
	kitchenAreaMultiplier:    50.0,
	maxKitchenArea:           50.0,
	defaultKitchenArea:       5.0,
	comissionMultiplier:      0.0006,
	prepaidMonthsMultiplier:  3000.0,
	petsMultiplier:           4000.0,
	dishwasherMultiplier:     1500.0,
	conditionerMultiplier:    1500.0,
	apartmentsMultiplier:     3000.0,
	twoRoomMultiplier:        3000.0,
	threeRoomMultiplier:      5000.0,
	fourRoomMultiplier:       7500.0,
	additionalRoomsMultipier: 1500.0,
	windowsYardMultiplier:    400.0,
	windowsStreetMultiplier:  200.0,
	windowsBothMultiplier:    500.0,
	renovationCosmeticMult:   3000.0,
	renovationEuroMult:       5500.0,
	renovationDesignMult:     7000.0,
	bathroomSeparatedMult:    1000.0,
	balconyMultiplier:        1500.0,
	loggiaMultiplier:         2500.0,
	defaultUndergroundScore:  50.0,
	undergroundScoreMult:     1000.0,
}

const saleScaleMultiplier = 200.0

var saleConstants = scoreConstants{
	defaultScore:             rentConstants.defaultScore * saleScaleMultiplier,
	depositMultiplier:        rentConstants.depositMultiplier,
	allAreaMultiplier:        rentConstants.allAreaMultiplier * saleScaleMultiplier,
	costMultiplier:           rentConstants.costMultiplier,
	maxPossiblePrice:         rentConstants.maxPossiblePrice * saleScaleMultiplier,
	livingAreaMultiplier:     rentConstants.livingAreaMultiplier * saleScaleMultiplier,
	livingAreaPart:           rentConstants.livingAreaPart,
	kitchenAreaMultiplier:    rentConstants.kitchenAreaMultiplier * saleScaleMultiplier,
	maxKitchenArea:           rentConstants.maxKitchenArea,
	defaultKitchenArea:       rentConstants.defaultKitchenArea,
	comissionMultiplier:      rentConstants.comissionMultiplier,
	prepaidMonthsMultiplier:  rentConstants.prepaidMonthsMultiplier,
	petsMultiplier:           rentConstants.petsMultiplier * saleScaleMultiplier,
	dishwasherMultiplier:     rentConstants.dishwasherMultiplier * saleScaleMultiplier,
	conditionerMultiplier:    rentConstants.conditionerMultiplier * saleScaleMultiplier,
	apartmentsMultiplier:     rentConstants.apartmentsMultiplier * saleScaleMultiplier,
	twoRoomMultiplier:        rentConstants.twoRoomMultiplier * saleScaleMultiplier,
	threeRoomMultiplier:      rentConstants.threeRoomMultiplier * saleScaleMultiplier,
	fourRoomMultiplier:       rentConstants.fourRoomMultiplier * saleScaleMultiplier,
	additionalRoomsMultipier: rentConstants.additionalRoomsMultipier * saleScaleMultiplier,
	windowsYardMultiplier:    rentConstants.windowsYardMultiplier * saleScaleMultiplier,
	windowsStreetMultiplier:  rentConstants.windowsStreetMultiplier * saleScaleMultiplier,
	windowsBothMultiplier:    rentConstants.windowsBothMultiplier * saleScaleMultiplier,
	renovationCosmeticMult:   rentConstants.renovationCosmeticMult * saleScaleMultiplier,
	renovationEuroMult:       rentConstants.renovationEuroMult * saleScaleMultiplier,
	renovationDesignMult:     rentConstants.renovationDesignMult * saleScaleMultiplier,
	bathroomSeparatedMult:    rentConstants.bathroomSeparatedMult * saleScaleMultiplier,
	balconyMultiplier:        rentConstants.balconyMultiplier * saleScaleMultiplier,
	loggiaMultiplier:         rentConstants.loggiaMultiplier * saleScaleMultiplier,
	defaultUndergroundScore:  rentConstants.defaultUndergroundScore,
	undergroundScoreMult:     rentConstants.undergroundScoreMult * saleScaleMultiplier,
}

const eps = 1e-5

// CustomParams holds the 18 customizable scoring multipliers a subscriber
// may override; field names match subscription_scoring_params columns.
type CustomParams struct {
	AllArea            float64
	KitchenArea        float64
	Pets               float64
	Dishwasher         float64
	Conditioner        float64
	Apartments         float64
	TwoRoom            float64
	ThreeRoom          float64
	FourRoom           float64
	AdditionalRooms    float64
	WindowsYard        float64
	WindowsStreet      float64
	WindowsBoth        float64
	RenovationDesign   float64
	RenovationEuro     float64
	RenovationCosmetic float64
	BathroomSeparated  float64
	Balcony            float64
	Loggia             float64
	Underground        float64
}

// Score computes a composite desirability score for a flat. If custom is
// nil, it reproduces exactly the default rent/sale formula (matching the
// FlatScore realty-parser already stored). If custom is set, its 18 fields
// override the corresponding base multipliers before scoring.
func Score(f *model.FlatInfo, custom *CustomParams) int {
	c := rentConstants
	if f.DealType == "sale" {
		c = saleConstants
	}
	if custom != nil {
		c.allAreaMultiplier = custom.AllArea
		c.kitchenAreaMultiplier = custom.KitchenArea
		c.petsMultiplier = custom.Pets
		c.dishwasherMultiplier = custom.Dishwasher
		c.conditionerMultiplier = custom.Conditioner
		c.apartmentsMultiplier = custom.Apartments
		c.twoRoomMultiplier = custom.TwoRoom
		c.threeRoomMultiplier = custom.ThreeRoom
		c.fourRoomMultiplier = custom.FourRoom
		c.additionalRoomsMultipier = custom.AdditionalRooms
		c.windowsYardMultiplier = custom.WindowsYard
		c.windowsStreetMultiplier = custom.WindowsStreet
		c.windowsBothMultiplier = custom.WindowsBoth
		c.renovationDesignMult = custom.RenovationDesign
		c.renovationEuroMult = custom.RenovationEuro
		c.renovationCosmeticMult = custom.RenovationCosmetic
		c.bathroomSeparatedMult = custom.BathroomSeparated
		c.balconyMultiplier = custom.Balcony
		c.loggiaMultiplier = custom.Loggia
		c.undergroundScoreMult = custom.Underground
	}
	return calculateScore(f, c)
}

func calculateScore(f *model.FlatInfo, c scoreConstants) int {
	score := c.defaultScore
	score -= c.depositMultiplier * float64(f.Deposit)
	score += c.allAreaMultiplier * f.TotalArea
	score += costScore(f, c)
	score += livingAreaScore(f, c)
	score += kitchenAreaScore(f, c)
	score -= c.comissionMultiplier * float64(f.Comission) * float64(f.Price)
	score -= c.prepaidMonthsMultiplier * float64(f.DepositMonths)
	if f.PetsAllowed {
		score += c.petsMultiplier
	}
	if f.HasDishwasher {
		score += c.dishwasherMultiplier
	}
	if f.HasConditioner {
		score += c.conditionerMultiplier
	}
	if f.IsApartments {
		score += c.apartmentsMultiplier
	}
	score += roomsScore(f, c)
	score += windowsScore(f, c)
	score += renovationScore(f, c)
	score += bathroomScore(f, c)
	score += math.Max(c.balconyMultiplier*float64(f.BalconyCount), c.loggiaMultiplier*float64(f.LoggiaCount))
	score += c.undergroundScoreMult * (c.defaultUndergroundScore - f.UndergroundScore)
	return int(score)
}

func costScore(f *model.FlatInfo, c scoreConstants) float64 {
	if f.Price == 0 || float64(f.Price) > c.maxPossiblePrice {
		return -c.costMultiplier * float64(f.Deposit)
	}
	return -c.costMultiplier * float64(f.Price)
}

func livingAreaScore(f *model.FlatInfo, c scoreConstants) float64 {
	if math.Abs(f.LivingArea) < eps {
		return c.livingAreaMultiplier * c.livingAreaPart * f.TotalArea
	}
	return c.livingAreaMultiplier * f.LivingArea
}

func kitchenAreaScore(f *model.FlatInfo, c scoreConstants) float64 {
	if math.Abs(f.KitchenArea) < eps || f.KitchenArea > c.maxKitchenArea {
		return c.kitchenAreaMultiplier * c.defaultKitchenArea
	}
	return c.kitchenAreaMultiplier * f.KitchenArea
}

func roomsScore(f *model.FlatInfo, c scoreConstants) float64 {
	switch {
	case f.RoomNumber <= 1:
		return 0
	case f.RoomNumber == 2:
		return c.twoRoomMultiplier
	case f.RoomNumber == 3:
		return c.threeRoomMultiplier
	case f.RoomNumber == 4:
		return c.fourRoomMultiplier
	default:
		return c.fourRoomMultiplier + float64(f.RoomNumber)*c.additionalRoomsMultipier
	}
}

func windowsScore(f *model.FlatInfo, c scoreConstants) float64 {
	switch f.WindowsView {
	case "yard":
		return c.windowsYardMultiplier
	case "street":
		return c.windowsStreetMultiplier
	case "":
		return 0
	default:
		return c.windowsBothMultiplier
	}
}

func renovationScore(f *model.FlatInfo, c scoreConstants) float64 {
	switch f.Renovation {
	case "cosmetic":
		return c.renovationCosmeticMult
	case "euro":
		return c.renovationEuroMult
	case "design":
		return c.renovationDesignMult
	default:
		return 0
	}
}

func bathroomScore(f *model.FlatInfo, c scoreConstants) float64 {
	return math.Max(0, c.bathroomSeparatedMult*float64(f.SeparatedBathroomCount))
}
