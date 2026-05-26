package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/db"
	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type planCreateInput struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	ImageURL     string `json:"image_url"`
	Price        int64  `json:"price" binding:"required"`
	DurationDays int    `json:"duration_days" binding:"required"`
	MaxDevices   int    `json:"max_devices" binding:"required"`
	Tag          string `json:"tag"`
	SortOrder    int    `json:"sort_order"`
	Active       *bool  `json:"active"`
}

type planUpdateInput struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	ImageURL     *string `json:"image_url"`
	Price        *int64  `json:"price"`
	DurationDays *int    `json:"duration_days"`
	MaxDevices   *int    `json:"max_devices"`
	Tag          *string `json:"tag"`
	SortOrder    *int    `json:"sort_order"`
	Active       *bool   `json:"active"`
}

func listPlansQuery() *gorm.DB {
	return db.DB.Order("sort_order ASC, id ASC")
}

func AddPlan(c *gin.Context) {
	var input planCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	active := true
	if input.Active != nil {
		active = *input.Active
	}

	plan := models.Plan{
		Name:         input.Name,
		Description:  input.Description,
		ImageURL:     input.ImageURL,
		Price:        input.Price,
		DurationDays: input.DurationDays,
		MaxDevices:   input.MaxDevices,
		Tag:          input.Tag,
		SortOrder:    input.SortOrder,
		Active:       active,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.DB.Create(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusCreated, plan)
}

func UpdatePlan(c *gin.Context) {
	var input planUpdateInput
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var plan models.Plan
	if err := db.DB.First(&plan, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	if input.Name != nil {
		plan.Name = *input.Name
	}
	if input.Description != nil {
		plan.Description = *input.Description
	}
	if input.ImageURL != nil {
		plan.ImageURL = *input.ImageURL
	}
	if input.Price != nil {
		plan.Price = *input.Price
	}
	if input.DurationDays != nil {
		plan.DurationDays = *input.DurationDays
	}
	if input.MaxDevices != nil {
		plan.MaxDevices = *input.MaxDevices
	}
	if input.Tag != nil {
		plan.Tag = *input.Tag
	}
	if input.SortOrder != nil {
		plan.SortOrder = *input.SortOrder
	}
	if input.Active != nil {
		plan.Active = *input.Active
	}

	plan.UpdatedAt = time.Now()

	if err := db.DB.Save(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, plan)
}

func DeletePlanWithID(c *gin.Context) {
	id := c.Param("id")

	result := db.DB.Delete(&models.Plan{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "message": "deleted"})
}

func GetPlans(c *gin.Context) {
	var plans []models.Plan
	q := listPlansPlans(c, listPlansQuery())
	if err := q.Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, plans)
}

func listPlansPlans(c *gin.Context, q *gorm.DB) *gorm.DB {
	// Public list hides inactive unless ?all=1 (admin)
	if c.Query("all") != "1" {
		q = q.Where("active = ?", true)
	}
	return q
}

func GetPlan(c *gin.Context) {
	id := c.Param("id")
	var plan models.Plan

	if err := db.DB.First(&plan, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	if !plan.Active && c.Query("all") != "1" {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	c.JSON(http.StatusOK, plan)
}

func AdminListPlans(c *gin.Context) {
	var plans []models.Plan
	if err := listPlansQuery().Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, plans)
}

func AdminReorderPlans(c *gin.Context) {
	var body struct {
		Order []struct {
			ID        uint `json:"id" binding:"required"`
			SortOrder int  `json:"sort_order"`
		} `json:"order" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, item := range body.Order {
		if err := db.DB.Model(&models.Plan{}).Where("id = ?", item.ID).Update("sort_order", item.SortOrder).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// ParsePlanID helper for tests
func ParsePlanID(id string) (uint, error) {
	n, err := strconv.ParseUint(id, 10, 64)
	return uint(n), err
}
