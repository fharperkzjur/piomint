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
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/loggers"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

var logger = loggers.GetLogger()

func InitRouter() *gin.Engine {

	logger = loggers.GetLogger()

	if !configs.GetAppConfig().Debug {
		gin.SetMode(gin.ReleaseMode)
	}else{
		gin.SetMode(gin.DebugMode)
	}
	//@todo: init common user identity authentication logic here
	r := gin.New()

	r.MaxMultipartMemory = 8 << 20 // 8 MiB

	r.GET("/swagger/*any", ginSwagger.DisablingWrapHandler(swaggerFiles.Handler, "DISABLE_SWAGGER"))

	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))

	r.Use(cors.Default())
	r.RedirectTrailingSlash = false
	//r.Use(Auth())

	//r.NoMethod(HandleNotFound)
	//r.NoRoute(HandleNotFound)

	r.Use(loggers.GinLogger(logger))
	r.Use(gin.Recovery())

    AddGroupAILab(r)
	AddGroupTraining(r)
	AddGroupAICode(r)
	AddGroupSysRuns(r)

	registerEndpoints()

	return r
}
