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
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/gin-gonic/gin"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
)

//normalize sort fields
func normalizeSorts(str string) (string,APIError) {
	var sorts string
	for _, v := range str {
		if v >= 'A' && v <= 'Z' {
			sorts += "_" + string(v+32)
		} else if v == '|'{
			sorts += " "
		} else if v >= 'a' && v <= 'z' || v == '_' || v == '-' || v == ','{
			sorts += string(v)
		} else{
			return "",exports.ParameterError("invalid sort fields!")
		}
	}
	return sorts,nil
}

func checkSearchCond(c*gin.Context,filters exports.QueryFilterMap) (cond*exports.SearchCond, err APIError){
	 cond = &exports.SearchCond{}
	 if err := c.ShouldBind(cond);err != nil {
		 return nil,exports.ParameterError(err.Error())
	 }
	 cond.Sort,err  =normalizeSorts(cond.Sort)
	 if err != nil{
	 	return
	 }
	 if cond.PageSize > 100 {
	 	cond.PageSize = 100
	 }else if cond.PageSize == 0 {
	 	cond.PageNum  = 0
	 }
	 if cond.PageNum >= 1{
	 	cond.Offset = (cond.PageNum - 1) * cond.PageSize
	 }
	 if len(filters) > 0 {
			cond.EqualFilters=make(map[string]string)
			cond.AdvanceOpFilters=make(map[string]interface{})
			for f,field := range filters {
				value := c.Query(f)
				if len(value) == 0 {
					value = c.Param(f)
				}
				if len(value) > 0 {
					if  field[0] == ':' {//@add: support advanced operator
						cond.AdvanceOpFilters[field[1:]]=value
					}else if field[1] == ':'{
						var v interface{}
						switch field[0]{
						  case 'd':   v,_ =strconv.ParseInt(value,0,64)
						  case 'u':   v,_ =strconv.ParseUint(value,0,64)
						  case 'f':   v,_ = strconv.ParseFloat(value,64)
						  case 't':   t,_ :=strconv.ParseInt(value,0,64)
						              v=time.Unix(t/1000,t%1000*1000000).UTC()
						  default:  log.Fatalf("invalid advance seach condition vlaue type:%c",field[0])
						}
						cond.AdvanceOpFilters[field[2:]] = v
					}else{
						cond.EqualFilters[field] = value
					}
				}
			}
	}

	return
}


func makePagedQueryResult(req* exports.SearchCond,data interface{},err APIError)(interface{},APIError) {
	totalPage := uint(0)
	if req.TotalCount > 0 && req.PageSize > 0 {
		totalPage = uint (math.Ceil( float64(req.TotalCount)/float64(req.PageSize)))
	}
	result := &exports.PagedResult{
		Total:      uint(req.TotalCount),
		TotalPages: totalPage,
		PageNum:    req.PageNum,
		PageSize:   req.PageSize,
		Next:       req.Next,
		Items:      data,
	}
	return result,err
}

func getUserToken(c*gin.Context) string{
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		return c.Query("token")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return ""
	}
	return parts[1]
}


