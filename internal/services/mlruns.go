package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

type APIError = exports.APIError

func GetEndpointUrl(mlrun*models.BasicMLRunContext) (interface{},APIError){
	 return "",exports.NotImplementError()
}
