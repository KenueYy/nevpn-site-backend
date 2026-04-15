package remna

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RemnaHandler struct {
	service IRemnaService
	logger  *slog.Logger
}

func NewHandler(remnaService IRemnaService, logger *slog.Logger) *RemnaHandler {
	logger.Info("Remna handler created")

	return &RemnaHandler{
		service: remnaService,
		logger:  logger,
	}
}

func (h *RemnaHandler) GetAllUsers(c *gin.Context) {
	h.logger.Info("RemnaHandler: GetAllUsers started")

	users, err := h.service.GetAllUsers(c.Request.Context())
	if err != nil {
		h.logger.Error("RemnaHandler: GetAllUsers failed",
			"error", err,
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("RemnaHandler: GetAllUsers succeeded",
		"users_count", len(users.Users),
	)
	c.JSON(http.StatusOK, users.Users)
}

func (h *RemnaHandler) GetUserByUUID(c *gin.Context) {
	id := c.Param("uuid")

	h.logger.Info("RemnaHandler: GetUserByUUID started",
		"uuid", id,
	)

	user, err := h.service.GetUserByUUID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("RemnaHandler: GetUserByUUID failed",
			"uuid", id,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("RemnaHandler: GetUserByUUID succeeded",
		"uuid", id,
	)
	c.JSON(http.StatusOK, user)
}

func (h *RemnaHandler) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")

	h.logger.Info("RemnaHandler: GetUserByEmail started",
		"email", email,
	)

	user, err := h.service.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		h.logger.Error("RemnaHandler: GetUserByEmail failed",
			"email", email,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("RemnaHandler: GetUserByEmail succeeded",
		"email", email,
	)
	c.JSON(http.StatusOK, user)
}

func (h *RemnaHandler) GetUserByTelegramID(c *gin.Context) {
	id := c.Param("id")

	h.logger.Info("RemnaHandler: GetUserByTelegramID started",
		"telegram_id", id,
	)

	user, err := h.service.GetUserByUUID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("RemnaHandler: GetUserByTelegramID failed",
			"telegram_id", id,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("RemnaHandler: GetUserByTelegramID succeeded",
		"telegram_id", id,
	)
	c.JSON(http.StatusOK, user)
}

func (h *RemnaHandler) UpdateUser(c *gin.Context) {
	h.logger.Info("RemnaHandler: UpdateUser started")

	var rUser RemnaUserRequest

	if err := c.ShouldBindJSON(&rUser); err != nil {
		h.logger.Warn("RemnaHandler: failed to bind UpdateUser request",
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("RemnaHandler: UpdateUser request validated",
		"uuid", rUser.UUID,
		"email", rUser.Email,
		"tag", rUser.Tag,
	)

	user, err := h.service.UpdateUser(c.Request.Context(), &rUser)
	if err != nil {
		h.logger.Error("RemnaHandler: UpdateUser failed",
			"uuid", rUser.UUID,
			"email", rUser.Email,
			"tag", rUser.Tag,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("RemnaHandler: UpdateUser succeeded",
		"uuid", user.UUID,
		"email", user.Email,
	)
	c.JSON(http.StatusOK, user)
}

func (h *RemnaHandler) CreateNewUser(c *gin.Context) {
	h.logger.Info("RemnaHandler: CreateNewUser started")

	var rUser RemnaUserRequest

	if err := c.ShouldBindJSON(&rUser); err != nil {
		h.logger.Warn("RemnaHandler: failed to bind CreateNewUser request",
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if rUser.ExpireAt == nil {
		h.logger.Warn("RemnaHandler: CreateNewUser validation failed",
			"reason", "expire_at is empty",
			"email", rUser.Email,
			"username", rUser.Username,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "ExpireAt is empty"})
		return
	}

	if rUser.Username == "" {
		h.logger.Warn("RemnaHandler: CreateNewUser validation failed",
			"reason", "username is empty",
			"email", rUser.Email,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is empty"})
		return
	}

	h.logger.Info("RemnaHandler: CreateNewUser request validated",
		"uuid", rUser.UUID,
		"email", rUser.Email,
		"username", rUser.Username,
		"tag", rUser.Tag,
	)

	user, err := h.service.CreateNewUser(c.Request.Context(), &rUser)
	if err != nil {
		h.logger.Error("RemnaHandler: CreateNewUser failed",
			"uuid", rUser.UUID,
			"email", rUser.Email,
			"username", rUser.Username,
			"tag", rUser.Tag,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("RemnaHandler: CreateNewUser succeeded",
		"uuid", user.UUID,
		"email", user.Email,
		"username", user.Username,
	)
	c.JSON(http.StatusOK, user)
}
