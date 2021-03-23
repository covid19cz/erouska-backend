package coviddata

type covidMetricsConfig struct {
	URL string `env:"UZIS_METRICS_URL, required"`
}

type vaccinationMetricsConfig struct {
	URL string `env:"UZIS_VACCINATION_METRICS_URL, required"`
}

type vaccinationDownloadRequest struct {
	Modified string            `json:"modified"`
	Source   string            `json:"source"`
	Data     []VaccinationData `json:"data"`
}

type covidDataDownloadRequest struct {
	Modified string       `json:"modified"`
	Source   string       `json:"source"`
	Data     []TotalsData `json:"data"`
}

// VaccinationData holds all the info about vaccinations per region/date
type VaccinationData struct {
	Date       string `json:"datum" validate:"required"`
	Vaccine    string `json:"vakcina" validate:"required"`
	RegionCode string `json:"kraj_nuts_kod validate:"required"`
	Region     string `json:"kraj_nazev" validate:"required"`
	AgeGroup   string `json:"vekova_skupina" validate:"required"`
	FirstDose  int    `json:"prvnich_davek" validate:"required"`
	SecondDose int    `json:"druhych_davek" validate:"required"`
	TotalDoses int    `json:"celkem_davek" validate:"required"`
}

// VaccinationsAggregatedData holds all the info about vaccinations aggregated by date
type VaccinationsAggregatedData struct {
	Modified        int64
	Date            string
	DailyFirstDose  int
	DailySecondDose int
	TotalFirstDose  int
	TotalSecondDose int
}

// TotalsData holds all the info about tests, cases and results
type TotalsData struct {
	Date                       string `json:"datum" validate:"required"`
	ActiveCasesTotal           int    `json:"aktivni_pripady"  validate:"required"`
	CuredTotal                 int    `json:"vyleceni"  validate:"required"`
	DeceasedTotal              int    `json:"umrti"  validate:"required"`
	CurrentlyHospitalizedTotal int    `json:"aktualne_hospitalizovani"  validate:"required"`
	TestsTotal                 int    // for backward compatibility
	TestsIncrease              int    // for backward compatibility
	TestsIncreaseDate          string // for backward compatibility
	ConfirmedCasesTotal        int    `json:"potvrzene_pripady_celkem"  validate:"required"`
	ConfirmedCasesIncrease     int    `json:"potvrzene_pripady_vcerejsi_den" validate:"required"`
	ConfirmedCasesIncreaseDate string `json:"potvrzene_pripady_vcerejsi_den_datum" validate:"required"`
	AntigenTestsTotal          int    `json:"provedene_antigenni_testy_celkem" validate:"required"`
	AntigenTestsIncrease       int    `json:"provedene_antigenni_testy_vcerejsi_den" validate:"required"`
	AntigenTestsIncreaseDate   string `json:"provedene_antigenni_testy_vcerejsi_den_datum" validate:"required"`
	PCRTestsTotal              int    `json:"provedene_testy_celkem" validate:"required"`
	PCRTestsIncrease           int    `json:"provedene_testy_vcerejsi_den" validate:"required"`
	PCRTestsIncreaseDate       string `json:"provedene_testy_vcerejsi_den_datum" validate:"required"`
	VaccinationsTotal          int    `json:"vykazana_ockovani_celkem" validate:"required"`
	VaccinationsIncrease       int    `json:"vykazana_ockovani_vcerejsi_den" validate:"required"`
	VaccinationsIncreaseDate   string `json:"vykazana_ockovani_vcerejsi_den_datum" validate:"required"`
}
