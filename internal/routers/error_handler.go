/* ******************************************************************************
* 2019 - present Contributed by Apulis Technology (Shenzhen) Co. LTD
*
* This program and the accompanying materials are made available under the
* terms of the MIT License, which is available at
* https://www.opensource.org/licenses/MIT
*
* See the NOTICE file distributed with this work for additional
* information regarding copyright ownership.
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
* WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
* License for the specific language governing permissions and limitations
* under the License.
*
* SPDX-License-Identifier: MIT
******************************************************************************/
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
