package handlers

import (
	"net/http"
	"strconv"

	"forgecrud-backend/shared/database"
	"forgecrud-backend/shared/database/models/notification"

	"github.com/gin-gonic/gin"
)

// @Summary Get all notifications
// @Description Get all notifications for current user
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} notification.Notification
// @Failure 500 {object} map[string]interface{}
// @Router /notifications [get]
func GetNotifications(c *gin.Context) {
	var notifications []notification.Notification

	db := database.GetDB()
	if err := db.Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// @Summary Get notification by ID
// @Description Get a specific notification by ID
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Notification ID"
// @Success 200 {object} notification.Notification
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /notifications/{id} [get]
func GetNotification(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	var notif notification.Notification
	db := database.GetDB()
	if err := db.First(&notif, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, notif)
}

// @Summary Create notification
// @Description Create a new notification
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param notification body notification.Notification true "Notification data"
// @Success 201 {object} notification.Notification
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /notifications [post]
func CreateNotification(c *gin.Context) {
	var notif notification.Notification

	if err := c.ShouldBindJSON(&notif); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	if err := db.Create(&notif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}

	c.JSON(http.StatusCreated, notif)
}

// @Summary Mark notification as read
// @Description Mark a notification as read
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Notification ID"
// @Success 200 {object} notification.Notification
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /notifications/{id}/read [put]
func MarkAsRead(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	var notif notification.Notification
	db := database.GetDB()
	
	if err := db.First(&notif, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	notif.IsRead = true
	if err := db.Save(&notif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification"})
		return
	}

	c.JSON(http.StatusOK, notif)
}

// @Summary Delete notification
// @Description Delete a notification by ID
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Notification ID"
// @Success 204
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /notifications/{id} [delete]
func DeleteNotification(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	db := database.GetDB()
	if err := db.Delete(&notification.Notification{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}

	c.Status(http.StatusNoContent)
}
