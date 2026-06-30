package response

import (
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/example/gin-api-scaffold/internal/apperr"
)

type ValidationDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			Error(c, apperr.PayloadTooLarge())
			return false
		}

		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			Error(c, apperr.BadRequestWithDetails(
				"validation_failed",
				"validation failed",
				validationDetails(dst, validationErrs),
			))
			return false
		}

		Error(c, apperr.BadRequest("invalid_request", err.Error()))
		return false
	}
	return true
}

func validationDetails(dst any, validationErrs validator.ValidationErrors) []ValidationDetail {
	details := make([]ValidationDetail, 0, len(validationErrs))
	fieldNames := jsonFieldNames(dst)
	for _, fieldErr := range validationErrs {
		field := fieldNames[fieldErr.StructField()]
		if field == "" {
			field = strings.ToLower(fieldErr.Field())
		}
		details = append(details, ValidationDetail{
			Field:  field,
			Reason: validationReason(fieldErr),
		})
	}
	return details
}

func jsonFieldNames(dst any) map[string]string {
	result := map[string]string{}
	value := reflect.ValueOf(dst)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return result
	}

	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return result
	}

	valueType := value.Type()
	for i := 0; i < valueType.NumField(); i++ {
		field := valueType.Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		if name != "-" {
			result[field.Name] = name
		}
	}
	return result
}

func validationReason(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "is required"
	case "email":
		return "invalid email"
	case "min":
		return "must be at least " + fieldErr.Param()
	case "max":
		return "must be at most " + fieldErr.Param()
	default:
		return "failed " + fieldErr.Tag() + " validation"
	}
}
