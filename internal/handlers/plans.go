package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/KenueYy/nevpn-site-backend/internal/db"
	"github.com/KenueYy/nevpn-site-backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AddPlan(c *gin.Context) {
	var plan models.Plan

	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	plan.CreatedAt = time.Now()
	plan.UpdatedAt = time.Now()

	if err := db.DB.Create(&plan).Error; err != nil {
		return
	}

	c.JSON(201, plan)

}

func UpdatePlan(c *gin.Context) {
	var input struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		ImageURL     string `json:"image_url"`
		Price        *int64 `json:"price"`
		DurationDays *int   `json:"duration_days"`
		MaxDevices   *int   `json:"max_devices"`
	}

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

	if input.Name != "" {
		plan.Name = input.Name
	}
	plan.Description = input.Description
	plan.ImageURL = input.ImageURL
	if input.Price != nil {
		plan.Price = *input.Price
	}
	if input.DurationDays != nil {
		plan.DurationDays = *input.DurationDays
	}
	if input.MaxDevices != nil {
		plan.MaxDevices = *input.MaxDevices
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
	var plan models.Plan

	if err := db.DB.Delete(&plan, "id=?", id).Error; err != nil {
		return
	}

	c.JSON(200, id)
}

func GetPlans(c *gin.Context) {
	var plans []models.Plan
	if err := db.DB.Find(&plans).Error; err != nil {
		return
	}

	c.JSON(http.StatusOK, plans)
}

func GetPlan(c *gin.Context) {
	var plan models.Plan
	id := c.Param("id")

	if err := db.DB.First(&plan, "id=?", id).Error; err != nil {
		return
	}

	c.JSON(http.StatusOK, plan)
}
