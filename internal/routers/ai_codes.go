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
	"github.com/apulis/bmod/ai-lab-backend/internal/services"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/gin-gonic/gin"
	"strings"
)

func AddGroupAICode(r *gin.Engine){

	rg := r.Group( exports.AILAB_API_VERSION +  "/repos")

	group := (*IAMRouteGroup)(rg)

	group.GET("/apps/*bind", wrapper(getAllRepos),  "repo:list")
	group.DELETE("/apps/*bind", wrapper(deleteRepo),"repo:unbind")
	group.GET("/:repo", wrapper(queryRepo),         "repo:view")
	group.PUT("/:repo", wrapper(updateRepo),        "repo:update")
	group.POST("", wrapper(createRepo),             "repo:bind")
}

func getAllRepos(c*gin.Context)(interface{},APIError){

	bind := c.Param("bind")
	bind = strings.TrimLeft(bind,"/")
	if len(bind) == 0 {
		return nil,exports.ParameterError("invalid app binding name")
	}
	cond,err := checkSearchCond(c,nil)
	if err != nil {
		return nil,err
	}
	var data interface{}
	data,err  = services.ListAppRepos(cond,bind,c.Query("extranet") == "1")
	return makePagedQueryResult(cond,data,err)
}

func queryRepo(c*gin.Context)(interface{},APIError){
	if repoId := c.Param("repo");len(repoId) > 0{
		return services.QueryAppRepoDetail(repoId,c.Query("extranet") == "1")
	}else{
		return nil,exports.ParameterError("invalid repo ID")
	}
}

func updateRepo(c*gin.Context)(interface{},APIError){
	return nil,exports.NotImplementError("update repo not implemented !!!")
}

func createRepo(c*gin.Context)(interface{},APIError){
	repo := &exports.ReqCreateRepo{}
	if err := c.ShouldBindJSON(repo);err != nil || len(repo.Bind) == 0 || len(repo.Creator) == 0 {
		return nil,exports.ParameterError("invalid create repo information !!!")
	}
	if repo.UserId == 0 {
		return nil,exports.ParameterError("invalid user id !!!")
	}
	if len(repo.Connector) == 0 {
		repo.Connector = exports.AICODE_CONNECTOR_GITEA
		repo.HttpUrl=""
		repo.SshUrl=""
	}
	return services.CreateAppRepoBind(repo)
}

func deleteRepo(c*gin.Context) (interface{},APIError){
	bind := c.Param("bind")
	bind = strings.TrimLeft(bind,"/")
	if  len(bind) > 0{
		return services.DeleteAppRepoBind(bind,c.Query("repoId"))
	}else{
		return nil,exports.ParameterError("invalid app binding name")
	}
}
