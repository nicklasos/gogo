package middleware

import (
	"strconv"

	"app/internal/errs"

	"github.com/gin-gonic/gin"
)

// CurrentUserID returns the authenticated user's ID. When there is none it answers 401
// itself and returns false, so a handler only has to return.
func CurrentUserID(c *gin.Context) (int32, bool) {
	userID, err := GetUserIDFromContext(c)
	if err != nil {
		errs.RespondWithError(c, err)
		return 0, false
	}
	return userID, true
}

// PathID parses a positive integer path parameter such as ":id". When it is not one it
// answers 400 itself and returns false.
func PathID(c *gin.Context, name string) (int32, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 32)
	if err != nil || id < 1 {
		errs.RespondWithBadRequest(c, errs.ErrKeyBadRequest, "Invalid "+name)
		return 0, false
	}
	return int32(id), true
}

// Page parses ?page and ?page_size with the defaults every list endpoint uses. When
// they are invalid it answers 400 itself and returns false.
func Page(c *gin.Context) (PaginationParams, bool) {
	params, err := GetPaginationParamsFromContext(c, 20, 1, 100)
	if err != nil {
		errs.RespondWithBadRequest(c, errs.ErrKeyBadRequest, err.Error())
		return params, false
	}
	return params, true
}
