package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent"
	"github.com/matthewbaird/ontology/ent/portfolio"
	"github.com/matthewbaird/ontology/ent/property"
	"github.com/matthewbaird/ontology/ent/schema"
	"github.com/matthewbaird/ontology/ent/unit"
	"github.com/matthewbaird/ontology/internal/types"
)

// PropertyHandler implements HTTP handlers for Portfolio, Property, and Unit.
type PropertyHandler struct {
	client *ent.Client
}

// NewPropertyHandler creates a new PropertyHandler.
func NewPropertyHandler(client *ent.Client) *PropertyHandler {
	return &PropertyHandler{client: client}
}

// ---------------------------------------------------------------------------
// Portfolio
// ---------------------------------------------------------------------------

type createPortfolioRequest struct {
	Name                    string   `json:"name"`
	ManagementType          string   `json:"management_type"`
	RequiresTrustAccounting bool     `json:"requires_trust_accounting"`
	Status                  string   `json:"status"`
	FiscalYearStartMonth    int      `json:"fiscal_year_start_month"`
	DefaultPaymentMethods   []string `json:"default_payment_methods,omitempty"`
	OwnerID                 string   `json:"owner_id"`
}

func (h *PropertyHandler) CreatePortfolio(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req createPortfolioRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	ownerID, err := uuid.Parse(req.OwnerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid owner_id")
		return
	}

	builder := h.client.Portfolio.Create().
		SetName(req.Name).
		SetManagementType(portfolio.ManagementType(req.ManagementType)).
		SetRequiresTrustAccounting(req.RequiresTrustAccounting).
		SetStatus(portfolio.Status(req.Status)).
		SetFiscalYearStartMonth(req.FiscalYearStartMonth).
		SetOwnerID(ownerID).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(portfolio.Source(audit.Source))

	if len(req.DefaultPaymentMethods) > 0 {
		builder.SetDefaultPaymentMethods(req.DefaultPaymentMethods)
	}
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	p, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *PropertyHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.client.Portfolio.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *PropertyHandler) ListPortfolios(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, err := h.client.Portfolio.Query().
		Limit(pg.Limit).Offset(pg.Offset).
		Order(ent.Asc(portfolio.FieldName)).
		All(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type updatePortfolioRequest struct {
	Name                  *string  `json:"name,omitempty"`
	ManagementType        *string  `json:"management_type,omitempty"`
	Status                *string  `json:"status,omitempty"`
	FiscalYearStartMonth  *int     `json:"fiscal_year_start_month,omitempty"`
	DefaultPaymentMethods []string `json:"default_payment_methods,omitempty"`
}

func (h *PropertyHandler) UpdatePortfolio(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req updatePortfolioRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.Portfolio.UpdateOneID(id).
		SetUpdatedBy(audit.Actor).
		SetSource(portfolio.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	if req.Name != nil {
		builder.SetName(*req.Name)
	}
	if req.ManagementType != nil {
		builder.SetManagementType(portfolio.ManagementType(*req.ManagementType))
	}
	if req.Status != nil {
		builder.SetStatus(portfolio.Status(*req.Status))
	}
	if req.FiscalYearStartMonth != nil {
		builder.SetFiscalYearStartMonth(*req.FiscalYearStartMonth)
	}
	if req.DefaultPaymentMethods != nil {
		builder.SetDefaultPaymentMethods(req.DefaultPaymentMethods)
	}

	p, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *PropertyHandler) ActivatePortfolio(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	p, err := h.client.Portfolio.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	if err := ValidateTransition(schema.ValidPortfolioTransitions, string(p.Status), "active"); err != nil {
		writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
		return
	}
	builder := h.client.Portfolio.UpdateOneID(id).
		SetStatus(portfolio.StatusActive).
		SetUpdatedBy(audit.Actor).
		SetSource(portfolio.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}
	updated, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// ---------------------------------------------------------------------------
// Property
// ---------------------------------------------------------------------------

type createPropertyRequest struct {
	Name                   string        `json:"name"`
	Address                types.Address `json:"address"`
	PropertyType           string        `json:"property_type"`
	Status                 string        `json:"status"`
	YearBuilt              int           `json:"year_built"`
	TotalSquareFootage     float64       `json:"total_square_footage"`
	TotalUnits             int           `json:"total_units"`
	LotSizeSqft            *float64      `json:"lot_size_sqft,omitempty"`
	Stories                *int          `json:"stories,omitempty"`
	ParkingSpaces          *int          `json:"parking_spaces,omitempty"`
	RentControlled         *bool         `json:"rent_controlled,omitempty"`
	CompliancePrograms     []string      `json:"compliance_programs,omitempty"`
	RequiresLeadDisclosure bool          `json:"requires_lead_disclosure"`
	InsuranceExpiry        *time.Time    `json:"insurance_expiry,omitempty"`
	PortfolioID            *string       `json:"portfolio_id,omitempty"`
}

func (h *PropertyHandler) CreateProperty(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req createPropertyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.Property.Create().
		SetName(req.Name).
		SetAddress(&req.Address).
		SetPropertyType(property.PropertyType(req.PropertyType)).
		SetStatus(property.Status(req.Status)).
		SetYearBuilt(req.YearBuilt).
		SetTotalSquareFootage(req.TotalSquareFootage).
		SetTotalUnits(req.TotalUnits).
		SetRequiresLeadDisclosure(req.RequiresLeadDisclosure).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(property.Source(audit.Source))

	if req.LotSizeSqft != nil {
		builder.SetNillableLotSizeSqft(req.LotSizeSqft)
	}
	if req.Stories != nil {
		builder.SetNillableStories(req.Stories)
	}
	if req.ParkingSpaces != nil {
		builder.SetNillableParkingSpaces(req.ParkingSpaces)
	}
	if req.RentControlled != nil {
		builder.SetRentControlled(*req.RentControlled)
	}
	if len(req.CompliancePrograms) > 0 {
		builder.SetCompliancePrograms(req.CompliancePrograms)
	}
	if req.InsuranceExpiry != nil {
		builder.SetNillableInsuranceExpiry(req.InsuranceExpiry)
	}
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	p, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *PropertyHandler) GetProperty(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.client.Property.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *PropertyHandler) ListProperties(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, err := h.client.Property.Query().
		Limit(pg.Limit).Offset(pg.Offset).
		Order(ent.Asc(property.FieldName)).
		All(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type updatePropertyRequest struct {
	Name                   *string        `json:"name,omitempty"`
	Address                *types.Address `json:"address,omitempty"`
	PropertyType           *string        `json:"property_type,omitempty"`
	Status                 *string        `json:"status,omitempty"`
	YearBuilt              *int           `json:"year_built,omitempty"`
	TotalSquareFootage     *float64       `json:"total_square_footage,omitempty"`
	TotalUnits             *int           `json:"total_units,omitempty"`
	LotSizeSqft            *float64       `json:"lot_size_sqft,omitempty"`
	Stories                *int           `json:"stories,omitempty"`
	ParkingSpaces          *int           `json:"parking_spaces,omitempty"`
	RentControlled         *bool          `json:"rent_controlled,omitempty"`
	CompliancePrograms     []string       `json:"compliance_programs,omitempty"`
	RequiresLeadDisclosure *bool          `json:"requires_lead_disclosure,omitempty"`
	InsuranceExpiry        *time.Time     `json:"insurance_expiry,omitempty"`
}

func (h *PropertyHandler) UpdateProperty(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req updatePropertyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.Property.UpdateOneID(id).
		SetUpdatedBy(audit.Actor).
		SetSource(property.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	if req.Name != nil {
		builder.SetName(*req.Name)
	}
	if req.Address != nil {
		builder.SetAddress(req.Address)
	}
	if req.PropertyType != nil {
		builder.SetPropertyType(property.PropertyType(*req.PropertyType))
	}
	if req.Status != nil {
		builder.SetStatus(property.Status(*req.Status))
	}
	if req.YearBuilt != nil {
		builder.SetYearBuilt(*req.YearBuilt)
	}
	if req.TotalSquareFootage != nil {
		builder.SetTotalSquareFootage(*req.TotalSquareFootage)
	}
	if req.TotalUnits != nil {
		builder.SetTotalUnits(*req.TotalUnits)
	}
	if req.LotSizeSqft != nil {
		builder.SetNillableLotSizeSqft(req.LotSizeSqft)
	}
	if req.Stories != nil {
		builder.SetNillableStories(req.Stories)
	}
	if req.ParkingSpaces != nil {
		builder.SetNillableParkingSpaces(req.ParkingSpaces)
	}
	if req.RentControlled != nil {
		builder.SetRentControlled(*req.RentControlled)
	}
	if req.CompliancePrograms != nil {
		builder.SetCompliancePrograms(req.CompliancePrograms)
	}
	if req.RequiresLeadDisclosure != nil {
		builder.SetRequiresLeadDisclosure(*req.RequiresLeadDisclosure)
	}
	if req.InsuranceExpiry != nil {
		builder.SetNillableInsuranceExpiry(req.InsuranceExpiry)
	}

	p, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *PropertyHandler) ActivateProperty(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	p, err := h.client.Property.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	if err := ValidateTransition(schema.ValidPropertyTransitions, string(p.Status), "active"); err != nil {
		writeError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
		return
	}
	builder := h.client.Property.UpdateOneID(id).
		SetStatus(property.StatusActive).
		SetUpdatedBy(audit.Actor).
		SetSource(property.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}
	updated, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// ---------------------------------------------------------------------------
// Unit
// ---------------------------------------------------------------------------

type createUnitRequest struct {
	UnitNumber           string   `json:"unit_number"`
	UnitType             string   `json:"unit_type"`
	Status               string   `json:"status"`
	SquareFootage        float64  `json:"square_footage"`
	Bedrooms             *int     `json:"bedrooms,omitempty"`
	Bathrooms            *float64 `json:"bathrooms,omitempty"`
	Floor                *int     `json:"floor,omitempty"`
	Amenities            []string `json:"amenities,omitempty"`
	FloorPlan            *string  `json:"floor_plan,omitempty"`
	ADAAccessible        *bool    `json:"ada_accessible,omitempty"`
	PetFriendly          *bool    `json:"pet_friendly,omitempty"`
	Furnished            *bool    `json:"furnished,omitempty"`
	MarketRentAmountCents *int64  `json:"market_rent_amount_cents,omitempty"`
	MarketRentCurrency   *string  `json:"market_rent_currency,omitempty"`
	PropertyID           *string  `json:"property_id,omitempty"`
}

func (h *PropertyHandler) CreateUnit(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req createUnitRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.Unit.Create().
		SetUnitNumber(req.UnitNumber).
		SetUnitType(unit.UnitType(req.UnitType)).
		SetStatus(unit.Status(req.Status)).
		SetSquareFootage(req.SquareFootage).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(unit.Source(audit.Source))

	if req.Bedrooms != nil {
		builder.SetNillableBedrooms(req.Bedrooms)
	}
	if req.Bathrooms != nil {
		builder.SetNillableBathrooms(req.Bathrooms)
	}
	if req.Floor != nil {
		builder.SetNillableFloor(req.Floor)
	}
	if len(req.Amenities) > 0 {
		builder.SetAmenities(req.Amenities)
	}
	if req.FloorPlan != nil {
		builder.SetNillableFloorPlan(req.FloorPlan)
	}
	if req.ADAAccessible != nil {
		builder.SetAdaAccessible(*req.ADAAccessible)
	}
	if req.PetFriendly != nil {
		builder.SetPetFriendly(*req.PetFriendly)
	}
	if req.Furnished != nil {
		builder.SetFurnished(*req.Furnished)
	}
	if req.MarketRentAmountCents != nil {
		builder.SetNillableMarketRentAmountCents(req.MarketRentAmountCents)
	}
	if req.MarketRentCurrency != nil {
		builder.SetNillableMarketRentCurrency(req.MarketRentCurrency)
	}
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	u, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (h *PropertyHandler) GetUnit(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	u, err := h.client.Unit.Get(r.Context(), id)
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *PropertyHandler) ListUnits(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	items, err := h.client.Unit.Query().
		Limit(pg.Limit).Offset(pg.Offset).
		Order(ent.Asc(unit.FieldUnitNumber)).
		All(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type updateUnitRequest struct {
	UnitNumber            *string  `json:"unit_number,omitempty"`
	UnitType              *string  `json:"unit_type,omitempty"`
	Status                *string  `json:"status,omitempty"`
	SquareFootage         *float64 `json:"square_footage,omitempty"`
	Bedrooms              *int     `json:"bedrooms,omitempty"`
	Bathrooms             *float64 `json:"bathrooms,omitempty"`
	Floor                 *int     `json:"floor,omitempty"`
	Amenities             []string `json:"amenities,omitempty"`
	FloorPlan             *string  `json:"floor_plan,omitempty"`
	ADAAccessible         *bool    `json:"ada_accessible,omitempty"`
	PetFriendly           *bool    `json:"pet_friendly,omitempty"`
	Furnished             *bool    `json:"furnished,omitempty"`
	MarketRentAmountCents *int64   `json:"market_rent_amount_cents,omitempty"`
	MarketRentCurrency    *string  `json:"market_rent_currency,omitempty"`
}

func (h *PropertyHandler) UpdateUnit(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "id")
	if !ok {
		return
	}
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}
	var req updateUnitRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	builder := h.client.Unit.UpdateOneID(id).
		SetUpdatedBy(audit.Actor).
		SetSource(unit.Source(audit.Source))
	if audit.CorrelationID != nil {
		builder.SetCorrelationID(*audit.CorrelationID)
	}

	if req.UnitNumber != nil {
		builder.SetUnitNumber(*req.UnitNumber)
	}
	if req.UnitType != nil {
		builder.SetUnitType(unit.UnitType(*req.UnitType))
	}
	if req.Status != nil {
		builder.SetStatus(unit.Status(*req.Status))
	}
	if req.SquareFootage != nil {
		builder.SetSquareFootage(*req.SquareFootage)
	}
	if req.Bedrooms != nil {
		builder.SetNillableBedrooms(req.Bedrooms)
	}
	if req.Bathrooms != nil {
		builder.SetNillableBathrooms(req.Bathrooms)
	}
	if req.Floor != nil {
		builder.SetNillableFloor(req.Floor)
	}
	if req.Amenities != nil {
		builder.SetAmenities(req.Amenities)
	}
	if req.FloorPlan != nil {
		builder.SetNillableFloorPlan(req.FloorPlan)
	}
	if req.ADAAccessible != nil {
		builder.SetAdaAccessible(*req.ADAAccessible)
	}
	if req.PetFriendly != nil {
		builder.SetPetFriendly(*req.PetFriendly)
	}
	if req.Furnished != nil {
		builder.SetFurnished(*req.Furnished)
	}
	if req.MarketRentAmountCents != nil {
		builder.SetNillableMarketRentAmountCents(req.MarketRentAmountCents)
	}
	if req.MarketRentCurrency != nil {
		builder.SetNillableMarketRentCurrency(req.MarketRentCurrency)
	}

	u, err := builder.Save(r.Context())
	if err != nil {
		entErrorToHTTP(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}
