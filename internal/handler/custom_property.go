// Custom property handlers — command implementations that go beyond
// standard CRUD operations generated from the CUE ontology.
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/matthewbaird/ontology/ent/property"
	"github.com/matthewbaird/ontology/ent/space"
	"github.com/matthewbaird/ontology/internal/event"
	"github.com/matthewbaird/ontology/internal/types"
)

// OnboardProperty creates a property with optional bulk space creation.
// This is the full onboarding command — goes beyond simple CreateProperty CRUD.
func (h *PropertyHandler) OnboardProperty(w http.ResponseWriter, r *http.Request) {
	audit, ok := parseAuditContext(w, r)
	if !ok {
		return
	}

	type spaceReq struct {
		SpaceNumber    string   `json:"space_number"`
		SpaceType      string   `json:"space_type"`
		Floor          *int     `json:"floor,omitempty"`
		SquareFootage  *float64 `json:"square_footage,omitempty"`
		Bedrooms       *int     `json:"bedrooms,omitempty"`
		Bathrooms      *float64 `json:"bathrooms,omitempty"`
		MarketRentCents *int64  `json:"market_rent_amount_cents,omitempty"`
		MarketRentCurrency string `json:"market_rent_currency,omitempty"`
	}
	type onboardReq struct {
		Name          string        `json:"name"`
		PortfolioID   string        `json:"portfolio_id"`
		Address       types.Address `json:"address"`
		PropertyType  string        `json:"property_type"`
		YearBuilt     int           `json:"year_built"`
		TotalSpaces   int           `json:"total_spaces"`
		Spaces        []spaceReq    `json:"spaces,omitempty"`
	}
	var req onboardReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "MISSING_NAME", "name is required")
		return
	}
	portfolioID, err := uuid.Parse(req.PortfolioID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid portfolio_id")
		return
	}
	if req.TotalSpaces < 1 {
		writeError(w, http.StatusBadRequest, "INVALID_SPACES", "total_spaces must be >= 1")
		return
	}

	tx, err := h.client.Tx(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TX_ERROR", err.Error())
		return
	}

	// Create property with onboarding status.
	propBuilder := tx.Property.Create().
		SetName(req.Name).
		SetAddress(&req.Address).
		SetPropertyType(property.PropertyType(req.PropertyType)).
		SetStatus(property.StatusOnboarding).
		SetYearBuilt(req.YearBuilt).
		SetTotalSquareFootage(0). // will be calculated from spaces
		SetTotalSpaces(req.TotalSpaces).
		SetRequiresLeadDisclosure(req.YearBuilt < 1978).
		SetPortfolioID(portfolioID).
		SetCreatedBy(audit.Actor).
		SetUpdatedBy(audit.Actor).
		SetSource(property.Source(audit.Source))
	if audit.CorrelationID != nil {
		propBuilder.SetCorrelationID(*audit.CorrelationID)
	}

	prop, err := propBuilder.Save(r.Context())
	if err != nil {
		tx.Rollback()
		entErrorToHTTP(w, err)
		return
	}

	// Create spaces if provided.
	var totalSqft float64
	for _, s := range req.Spaces {
		sqft := float64(0)
		if s.SquareFootage != nil {
			sqft = *s.SquareFootage
		}
		totalSqft += sqft

		spBuilder := tx.Space.Create().
			SetSpaceNumber(s.SpaceNumber).
			SetSpaceType(space.SpaceType(s.SpaceType)).
			SetStatus(space.StatusVacant).
			SetLeasable(true).
			SetSquareFootage(sqft).
			SetPropertyID(prop.ID).
			SetCreatedBy(audit.Actor).
			SetUpdatedBy(audit.Actor).
			SetSource(space.Source(audit.Source))
		if s.Floor != nil {
			spBuilder.SetFloor(*s.Floor)
		}
		if s.Bedrooms != nil {
			spBuilder.SetBedrooms(*s.Bedrooms)
		}
		if s.Bathrooms != nil {
			spBuilder.SetBathrooms(*s.Bathrooms)
		}
		if s.MarketRentCents != nil {
			spBuilder.SetMarketRentAmountCents(*s.MarketRentCents)
			curr := s.MarketRentCurrency
			if curr == "" {
				curr = "USD"
			}
			spBuilder.SetMarketRentCurrency(curr)
		}
		if _, err := spBuilder.Save(r.Context()); err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}
	}

	// Update total square footage if spaces were provided.
	if len(req.Spaces) > 0 && totalSqft > 0 {
		_, err = tx.Property.UpdateOneID(prop.ID).
			SetTotalSquareFootage(totalSqft).
			Save(r.Context())
		if err != nil {
			tx.Rollback()
			entErrorToHTTP(w, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "COMMIT_ERROR", err.Error())
		return
	}

	recordEvent(r.Context(), event.NewPropertyOnboarded(event.PropertyOnboardedPayload{
		PropertyID:   prop.ID.String(),
		PortfolioID:  req.PortfolioID,
		PropertyType: req.PropertyType,
		Address:      req.Address,
		SpaceCount:   len(req.Spaces),
	}))

	writeJSON(w, http.StatusCreated, prop)
}
