package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"sovera-core-api/internal/model"
	"sovera-core-api/internal/service/keyperson"
)

type KeyPersonHandler struct {
	service *keyperson.KeyPersonService
}

func NewKeyPersonHandler(service *keyperson.KeyPersonService) *KeyPersonHandler {
	return &KeyPersonHandler{
		service: service,
	}
}

// ListKeyPersons returns all key persons for a given company ID.
// GET /api/v1/companies/:id/key-persons
func (h *KeyPersonHandler) ListKeyPersons(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_INPUT",
			"message": "Company ID parameter is required",
		})
	}

	persons, err := h.service.GetKeyPersonsByCompanyID(c.Context(), companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    persons,
	})
}

// CreateKeyPerson adds a new key person for a company.
// POST /api/v1/companies/:id/key-persons
func (h *KeyPersonHandler) CreateKeyPerson(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_INPUT",
			"message": "Company ID parameter is required",
		})
	}

	var req model.CompanyKeyPerson
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": err.Error(),
		})
	}

	req.CompanyID = companyID

	created, err := h.service.CreateKeyPerson(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "CREATE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    created,
	})
}

// UpdateKeyPerson updates an existing key person profile.
// PUT /api/v1/key-persons/:id
func (h *KeyPersonHandler) UpdateKeyPerson(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_INPUT",
			"message": "Key Person ID is required",
		})
	}

	var req model.CompanyKeyPerson
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": err.Error(),
		})
	}

	req.ID = id

	updated, err := h.service.UpdateKeyPerson(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "UPDATE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    updated,
	})
}

// DeleteKeyPerson removes a key person.
// DELETE /api/v1/key-persons/:id
func (h *KeyPersonHandler) DeleteKeyPerson(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_INPUT",
			"message": "Key Person ID is required",
		})
	}

	if err := h.service.DeleteKeyPerson(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "DELETE_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Key person deleted successfully",
	})
}

// ListSocialSignals returns social/news signals for a company.
// GET /api/v1/companies/:id/key-person-signals
func (h *KeyPersonHandler) ListSocialSignals(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_INPUT",
			"message": "Company ID parameter is required",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	actionableOnly := c.Query("actionable", "") == "true"

	if actionableOnly {
		signals, err := h.service.GetActionableSignals(c.Context(), companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "QUERY_FAILED",
				"message": err.Error(),
			})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    signals,
		})
	}

	signals, err := h.service.GetSocialSignalsByCompanyID(c.Context(), companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "QUERY_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    signals,
	})
}

// IngestSocialSignal ingests a social or news signal for a company.
// POST /api/v1/companies/:id/key-person-signals
func (h *KeyPersonHandler) IngestSocialSignal(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_INPUT",
			"message": "Company ID parameter is required",
		})
	}

	var req model.KeyPersonSocialSignal
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INVALID_PAYLOAD",
			"message": err.Error(),
		})
	}

	req.CompanyID = companyID

	ingested, err := h.service.IngestSocialSignal(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "INGESTION_FAILED",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    ingested,
	})
}
