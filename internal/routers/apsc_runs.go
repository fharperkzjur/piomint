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
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/gin-gonic/gin"
)



func AddGroupSysRuns(r*gin.Engine){
 	  rg := r.Group(exports.AILAB_API_VERSION + "/job-management/:app")

	  group := (*IAMRouteGroup)(rg)

	  group.GET("/jobs", wrapper(sysGetRunList),       "apsc-run:list")
	  group.GET("/metaInfo",wrapper(sysGetRunMetaInfo),"apsc-meta:view")
	  group.POST("/:runId/stop",wrapper(sysKillRun),   "apsc-run:kill")
}


func sysGetRunList(c*gin.Context) (interface{},APIError){

	  cond,err := checkSearchCond(c,exports.QueryFilterMap{
			"jobType":     "job_type",
		    "jobStatus" :  "status",
		    "orgId" :      "org_id",
		    "userGroupId": "user_group_id",
		    "projectName": ":project_name like ? ",
			"userName" :   "runs.creator",
			"startTime":   "t:start_time >= ?",
			"endTime"  :   "t:end_time <= ?",
			"app" : "app",
		})
	  if err != nil {
			return nil,err
	  }
	  //@add: default sort by created_at desc
	  if len(cond.Sort) == 0 {
		cond.Sort = "runs.created_at desc"
	  }
	  var data interface{}
	  data,err  = models.SysGetAllRuns(cond)
	  return makePagedQueryResult(cond,data,err)
}

func sysGetRunMetaInfo(c*gin.Context) (interface{},APIError){
	 taskTypes:= make([]map[string]interface{},0,len(exports.AILAB_JOB_TYPES_ZH))
	 for k,v := range(exports.AILAB_JOB_TYPES_ZH) {
	 	taskTypes=append(taskTypes,map[string]interface{}{
		    "label": v ,"value": k,
	    })
	 }
	 taskStatus := []map[string]interface{}{
		{"label": "初始化" ,"value": exports.AILAB_RUN_STATUS_INIT },
		{"label": "启动中" ,"value": exports.AILAB_RUN_STATUS_STARTING },
		{"label": "队列中" ,"value": exports.AILAB_RUN_STATUS_QUEUE },
		{"label": "调度中" ,"value": exports.AILAB_RUN_STATUS_SCHEDULE },
		{"label": "关闭中" ,"value": exports.AILAB_RUN_STATUS_KILLING },
		{"label": "结束中" ,"value": exports.AILAB_RUN_STATUS_STOPPING },
		{"label": "运行" ,"value": exports.AILAB_RUN_STATUS_RUN },
		{"label": "等待子任务" ,"value": exports.AILAB_RUN_STATUS_WAIT_CHILD },
		{"label": "保存中" ,"value": exports.AILAB_RUN_STATUS_COMPLETING },
		{"label": "成功" ,"value": exports.AILAB_RUN_STATUS_SUCCESS },
		{"label": "终止" ,"value": exports.AILAB_RUN_STATUS_ABORT },
		{"label": "错误" ,"value": exports.AILAB_RUN_STATUS_ERROR },
		{"label": "失败" ,"value": exports.AILAB_RUN_STATUS_FAIL },
		{"label": "保存失败" ,"value": exports.AILAB_RUN_STATUS_SAVE_FAIL },
	}
	return map[string]interface{}{
		"taskTypes":taskTypes,
		"taskStatus":taskStatus,
	},nil
}

func sysKillRun(c*gin.Context) (interface{},APIError){
	 runId := c.Param("runId")
	 if len(runId) == 0 {
	 	 return nil,exports.ParameterError("invalid run id !")
	 }
	 return models.KillLabRun(0,runId,false)
}
