
package routers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

type APIError = exports.APIError

//@modify:  simple handle request wrapper , always return data , error information
type RequestProcessor func(c *gin.Context) (interface{},APIError)

func wrapper(handler RequestProcessor) func(c *gin.Context) {
	return func(c *gin.Context) {

		data,err := handler(c)
		resp := exports.CommResponse{Data: data}

		statusCode := http.StatusOK
		if err != nil {//process for error
			logger.Error(err.Error())

			resp.Code=err.Errno()
			resp.Msg =err.Error()

			if h, ok := err.(*exports.APIException); ok {
				statusCode=h.StatusCode
			}
		}
		c.JSON(statusCode,resp)
	}
}
