package users

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *UserService
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{service: NewUserService(db)}
}

// GetUser godoc
// @Summary      Get user by ID
// @Description  Get a user by their ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  User
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser godoc
// @Summary      Create a new user
// @Description  Create a new user with name and email
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user  body      CreateUserParams  true  "User data"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Router       /users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	var params CreateUserParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), params.Name, params.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":    user,
		"message": "User created successfully",
	})
}

// ListUsers godoc
// @Summary      List all users
// @Description  Get a list of all users
// @Tags         users
// @Accept       json
// @Produce      json
// @Success      200  {array}   User
// @Failure      500  {object}  map[string]string
// @Router       /users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.service.GetAllUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}