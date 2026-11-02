package main

import (
	"time"

	"github.com/DextaAfrica/Backend/internal/config"
	"github.com/DextaAfrica/Backend/internal/domain/identity"
	"github.com/DextaAfrica/Backend/internal/domain/property"
	"github.com/DextaAfrica/Backend/internal/infrastructure/database"
	"github.com/DextaAfrica/Backend/internal/infrastructure/logger"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	logger.Init("development")
	logger.Log.Info().Msg("Starting Dexta Africa Enterprise Seeder...")

	db, err := database.Connect(&cfg.Database)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Database connection failed during seeding")
	}

	if err := db.AutoMigrate(); err != nil {
		logger.Log.Fatal().Err(err).Msg("Auto-migration failed during seeding")
	}

	seedUsers(db)
	seedDevelopments(db)

	logger.Log.Info().Msg("Database seeding completed successfully with realistic continental portfolio data!")
}

func seedUsers(db *database.PostgresDB) {
	var count int64
	db.DB.Model(&identity.User{}).Count(&count)
	if count > 0 {
		logger.Log.Info().Msg("Users table already populated, skipping user seed.")
		return
	}

	superAdminPassword, _ := bcrypt.GenerateFromPassword([]byte("DextaAdmin2026!"), bcrypt.DefaultCost)
	agentPassword, _ := bcrypt.GenerateFromPassword([]byte("DextaAgent2026!"), bcrypt.DefaultCost)

	users := []identity.User{
		{
			Email:        "admin@dextaafrica.com",
			PasswordHash: string(superAdminPassword),
			FirstName:    "Executive",
			LastName:     "Director",
			PhoneNumber:  "+2348000000001",
			Role:         identity.RoleSuperAdmin,
			Status:       identity.StatusActive,
			CountryCode:  "NG",
		},
		{
			Email:        "operations@dextaafrica.com",
			PasswordHash: string(superAdminPassword),
			FirstName:    "Operations",
			LastName:     "Lead",
			PhoneNumber:  "+2348000000002",
			Role:         identity.RoleOperationsDirector,
			Status:       identity.StatusActive,
			CountryCode:  "NG",
		},
		{
			Email:        "realtor.lagos@dextaafrica.com",
			PasswordHash: string(agentPassword),
			FirstName:    "Amara",
			LastName:     "Okonkwo",
			PhoneNumber:  "+2348000000003",
			Role:         identity.RoleRealtorAgent,
			Status:       identity.StatusActive,
			CountryCode:  "NG",
		},
	}

	for _, u := range users {
		db.DB.Create(&u)
	}
	logger.Log.Info().Int("count", len(users)).Msg("Seeded enterprise users")
}

func seedDevelopments(db *database.PostgresDB) {
	var count int64
	db.DB.Model(&property.Development{}).Count(&count)
	if count > 0 {
		logger.Log.Info().Msg("Developments already seeded, skipping.")
		return
	}

	completionDate1 := time.Now().AddDate(1, 4, 0)
	completionDate2 := time.Now().AddDate(0, 8, 0)

	dev1 := property.Development{
		Name:           "Dexta Pinnacle Waterfront Residences",
		Slug:           "dexta-pinnacle-waterfront-residences",
		Tagline:        "Ultra-Luxury Continental Living on the Atlantic Shoreline",
		Description:    "An iconic 24-storey high-rise development featuring uninterrupted ocean panoramas, private yacht slips, infinity pools, biometric smart security, and bespoke Italian interior finishes.",
		Country:        "Nigeria",
		StateProvince:  "Lagos",
		City:           "Victoria Island",
		Address:        "12 Maritime Crescent, Eko Atlantic City",
		Latitude:       6.4253,
		Longitude:      3.4095,
		Status:         property.DevStatusUnderConstruction,
		TotalUnits:     48,
		AvailableUnits: 36,
		StartingPrice:  450000,
		Currency:       "USD",
		IsFeatured:     true,
		CompletionDate: &completionDate1,
	}
	db.DB.Create(&dev1)

	// Seed Units for Dev1
	unitsDev1 := []property.Unit{
		{
			DevelopmentID:      dev1.ID,
			UnitNumber:         "P-0402",
			Title:              "The Azure 2-Bedroom Panorama Suite",
			Type:               property.UnitTypeApartment,
			Bedrooms:           2,
			Bathrooms:          2.5,
			FloorAreaSqm:       165.5,
			Price:              450000,
			Currency:           "USD",
			Status:             property.UnitStatusAvailable,
			ReservationDeposit: 25000,
			FloorNumber:        4,
		},
		{
			DevelopmentID:      dev1.ID,
			UnitNumber:         "P-1201",
			Title:              "The Grand Atlantic 3-Bedroom Penthouse Unit",
			Type:               property.UnitTypePenthouse,
			Bedrooms:           3,
			Bathrooms:          3.5,
			FloorAreaSqm:       285.0,
			Price:              820000,
			Currency:           "USD",
			Status:             property.UnitStatusAvailable,
			ReservationDeposit: 50000,
			FloorNumber:        12,
		},
		{
			DevelopmentID:      dev1.ID,
			UnitNumber:         "P-1801",
			Title:              "The Presidential Sky Mansion Duplex",
			Type:               property.UnitTypeMansion,
			Bedrooms:           5,
			Bathrooms:          6.0,
			FloorAreaSqm:       520.0,
			Price:              1750000,
			Currency:           "USD",
			Status:             property.UnitStatusAvailable,
			ReservationDeposit: 100000,
			FloorNumber:        18,
		},
	}
	for _, u := range unitsDev1 {
		db.DB.Create(&u)
	}

	// Seed Milestones for Dev1
	targetFoundation := time.Now().AddDate(0, -3, 0)
	targetRoofing := time.Now().AddDate(0, 4, 0)
	milestones := []property.Milestone{
		{
			DevelopmentID:      dev1.ID,
			Title:              "Substructure & Piling Completion",
			Description:        "Bored piling and deep marine-grade concrete foundations completed and passed stress-testing.",
			PercentageComplete: 100,
			TargetDate:         &targetFoundation,
			CompletedDate:      &targetFoundation,
			ProofImageURL:      "https://images.unsplash.com/photo-1541888946425-d0fbb18615f3",
		},
		{
			DevelopmentID:      dev1.ID,
			Title:              "Superstructure & Core Flooring",
			Description:        "Level 16 reinforced concrete slab cast. Core elevator shafts advanced.",
			PercentageComplete: 65,
			TargetDate:         &targetRoofing,
			ProofImageURL:      "https://images.unsplash.com/photo-1503387762-592deb58ef4e",
		},
	}
	for _, m := range milestones {
		db.DB.Create(&m)
	}

	// Seed Dev2 (Nairobi, Kenya)
	dev2 := property.Development{
		Name:           "Dexta Serena Heights",
		Slug:           "dexta-serena-heights-nairobi",
		Tagline:        "Boutique Urban Sanctuary Overlooking Karura Forest",
		Description:    "Nestled in prime Muthaiga/Gigiri diplomatic enclave, Serena Heights offers solar-powered villas with private courtyards, geothermal heating, and smart home automation.",
		Country:        "Kenya",
		StateProvince:  "Nairobi",
		City:           "Nairobi",
		Address:        "Limuru Road, Gigiri Zone",
		Latitude:       -1.2329,
		Longitude:      36.8197,
		Status:         property.DevStatusUnderConstruction,
		TotalUnits:     20,
		AvailableUnits: 14,
		StartingPrice:  380000,
		Currency:       "USD",
		IsFeatured:     true,
		CompletionDate: &completionDate2,
	}
	db.DB.Create(&dev2)

	unitsDev2 := []property.Unit{
		{
			DevelopmentID:      dev2.ID,
			UnitNumber:         "V-01",
			Title:              "The Acacia Forest Villa",
			Type:               property.UnitTypeVilla,
			Bedrooms:           4,
			Bathrooms:          4.5,
			FloorAreaSqm:       340.0,
			Price:              520000,
			Currency:           "USD",
			Status:             property.UnitStatusAvailable,
			ReservationDeposit: 30000,
		},
		{
			DevelopmentID:      dev2.ID,
			UnitNumber:         "V-05",
			Title:              "The Karura Garden Villa",
			Type:               property.UnitTypeVilla,
			Bedrooms:           4,
			Bathrooms:          4.5,
			FloorAreaSqm:       360.0,
			Price:              560000,
			Currency:           "USD",
			Status:             property.UnitStatusAvailable,
			ReservationDeposit: 30000,
		},
	}
	for _, u := range unitsDev2 {
		db.DB.Create(&u)
	}

	logger.Log.Info().Msg("Seeded continental developments, units, and construction milestones")
}

