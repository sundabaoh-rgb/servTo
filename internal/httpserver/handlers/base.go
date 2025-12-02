package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/response"
)

var validate = validator.New()

type BaseHandler struct{}

func (h *BaseHandler) BindJSON(w http.ResponseWriter, r *http.Request, dest interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		response.BadRequest(w, "invalid json format")
		return false
	}

	// Валидация структуры если есть теги validate
	if err := validate.Struct(dest); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			// Преобразуем ошибки валидации в читаемый формат
			errorMsg := "validation failed: "
			for _, e := range validationErrors {
				errorMsg += e.Field() + " " + e.Tag() + "; "
			}
			response.BadRequest(w, errorMsg)
		} else {
			response.BadRequest(w, "validation error")
		}
		return false
	}

	return true
}

func (h *BaseHandler) RequireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	return true
}
