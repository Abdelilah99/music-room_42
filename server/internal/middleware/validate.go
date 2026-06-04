package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// RegisterJSONTagNames configures gin's validator to report field errors using
// the json tag name (e.g. "email") instead of the Go field name (e.g. "Email"),
// so clients see field names that match what they sent.
func RegisterJSONTagNames() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	}
}

// BindAndValidate binds the JSON request body into obj and runs validation.
// On failure it writes a 400 response with a human-readable field list and
// returns false — callers should return immediately when false is returned.
func BindAndValidate(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			fields := make([]string, len(ve))
			for i, fe := range ve {
				fields[i] = describeFieldError(fe)
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "validation failed",
				"fields": fields,
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		}
		return false
	}
	return true
}

func describeFieldError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", fe.Field())
	default:
		return fmt.Sprintf("%s is invalid (%s)", fe.Field(), fe.Tag())
	}
}
