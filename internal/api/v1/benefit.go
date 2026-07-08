package v1

import (
	"net/http"

	ierr "github.com/flexprice/flexprice/internal/errors"
	"github.com/flexprice/flexprice/internal/logger"
	"github.com/flexprice/flexprice/internal/service"
	"github.com/gin-gonic/gin"
)

type BenefitHandler struct {
	benefitService service.BenefitService
	log            *logger.Logger
}

func NewBenefitHandler(benefitService service.BenefitService, log *logger.Logger) *BenefitHandler {
	return &BenefitHandler{
		benefitService: benefitService,
		log:            log,
	}
}

// GetBenefits godoc
// @Summary Get aggregated benefits for a customer and SKU
// @ID getBenefits
// @Description Returns lifetime benefits granted to a customer for a SKU, aggregated by feature from the benefit ledger.
// @Tags Benefits
// @Produce json
// @Security ApiKeyAuth
// @Param external_customer_id query string true "External customer ID"
// @Param sku query string true "SKU"
// @Success 200 {array} dto.BenefitAggregateResponse
// @Failure 400 {object} ierr.ErrorResponse "Invalid request"
// @Failure 404 {object} ierr.ErrorResponse "Customer not found"
// @Failure 500 {object} ierr.ErrorResponse "Server error"
// @Router /benefits [get]
func (h *BenefitHandler) GetBenefits(c *gin.Context) {
	externalCustomerID := c.Query("external_customer_id")
	sku := c.Query("sku")

	if externalCustomerID == "" || sku == "" {
		c.Error(ierr.NewError("external_customer_id and sku are required").
			WithHint("Provide external_customer_id and sku query params").
			Mark(ierr.ErrValidation))
		return
	}

	benefits, err := h.benefitService.GetBenefitsBySKU(c.Request.Context(), externalCustomerID, sku)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, benefits)
}
