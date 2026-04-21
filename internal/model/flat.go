package model

type FlatInfo struct {
	Link                     string  `json:"Link"`
	Price                    int     `json:"Price"`
	FlatScore                int     `json:"FlatScore"`
	UndergroundScore         float64 `json:"UndergroundScore"`
	UndergroundPlace         int     `json:"UndergroundPlace"`
	UndergroundDistanceInfo  string  `json:"UndergroundDistanceInfo"`
	RoomNumber               int     `json:"RoomNumber"`
	TotalArea                float64 `json:"TotalArea"`
	LivingArea               float64 `json:"LivingArea"`
	KitchenArea              float64 `json:"KitchenArea"`
	Floor                    int     `json:"Floor"`
	MaxFloor                 int     `json:"MaxFloor"`
	Deposit                  int     `json:"Deposit"`
	DepositMonths            int     `json:"DepositMonths"`
	Comission                int     `json:"Comission"`
	Renovation               string  `json:"Renovation"`
	IsApartments             bool    `json:"IsApartments"`
	LoggiaCount              int     `json:"LoggiaCount"`
	BalconyCount             int     `json:"BalconyCount"`
	WindowsView              string  `json:"WindowsView"`
	SeparatedBathroomCount   int     `json:"SeparatedBathroomCount"`
	CombinedBathroomCount    int     `json:"CombinedBathroomCount"`
	HasDishwasher            bool    `json:"HasDishwasher"`
	HasConditioner           bool    `json:"HasConditioner"`
	ChildrenAllowed          bool    `json:"ChildrenAllowed"`
	PetsAllowed              bool    `json:"PetsAllowed"`
	LastUpdated              string  `json:"LastUpdated"`
	CeilingHeight            float64 `json:"CeilingHeight"`
	BuildingEntrancesNumber  int     `json:"BuildingEntrancesNumber"`
	BuildingApartmentsNumber int     `json:"BuildingApartmentsNumber"`
	BuildingElevatorsNumber  int     `json:"BuildingElevatorsNumber"`
}
