package routers

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/gin-gonic/gin"
	"strconv"
)

func AddGroupTraining(r*gin.Engine){

	group := r.Group("/api/v1/labs/:lab")

	group.POST("/runs", wrapper(submitLabRun))
	group.POST("/:runId/evaluations", wrapper(submitLabEvaluate))
    // operates on lab runs : open_visual/close_visual  , pause|resume|kill|stop
	group.POST("/runs/:runId", wrapper(postLabRuns))
    // support train&evaluate job list
	group.GET("/runs", wrapper(getAllLabRuns))
	group.GET("/runs/:runId",wrapper(queryLabRun))
	group.GET("/runs/stats",wrapper(queryLabRunStats))

	group.DELETE("/runs/:runId", wrapper(delLabRun))
    // following interfce should only be called by admin role users
	group = r.Group("/api/v1/runs")
	group.GET("/",wrapper(sysGetAllLabRuns))
	group.GET("/:runId",wrapper(sysQueryLabRun))
	group.GET("/stats",wrapper(sysQueryLabRunStats))
	group.POST("/:runId", wrapper(sysPostLabRuns))

	group.POST("/clean",wrapper(sysCleanLabRuns))
	group.POST("/clean-stgy",wrapper(sysResetCleanStrategy))
}

func submitLabRun(c*gin.Context) (interface{},APIError){
	 labId, _ := parseLabRunId(c)
	 if labId == 0 {
	 	return nil,exports.ParameterError("create run invalid lab id")
	 }
	 req := &exports.CreateJobRequest{}
	 if err := c.ShouldBindJSON(req);err != nil{
	 	return nil,exports.ParameterError("invalid json data")
	 }
	 req.JobType = exports.AILAB_RUN_TRAINING
	 return models.CreateLabRun(labId,"",req)
}

func submitLabEvaluate(c*gin.Context)(interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("create nest run invalid lab id or run id")
	}
	req := &exports.CreateJobRequest{}
	if err := c.ShouldBindJSON(req);err != nil{
		return nil,exports.ParameterError("invalid json data")
	}
	req.JobType = exports.AILAB_RUN_EVALUATE
	return models.CreateLabRun(labId,runId,req)
}

func saveLabRun(labId uint64, runId string,req *exports.CreateJobRequest) (interface{},APIError){
	req.JobType  = exports.AILAB_RUN_SAVE
	req.JobFlags = exports.RUN_FLAGS_SINGLE_INSTANCE | exports.RUN_FLAGS_AUTO_DELETED | exports.RUN_FLAGS_RESUMEABLE
	_, err := models.CreateLabRun(labId,runId,req)
	if err == nil {// created new run
		return nil,exports.RaiseAPIError(exports.AILAB_WOULD_BLOCK,"wait to start visual job ...")
	}else if err.Errno() == exports.AILAB_SINGLETON_RUN_EXISTS {// exists old job
		_, err := models.ResumeLabRun(labId,err.Error())
		if err == nil{//@todo: here should return visit url

		}
		return nil,err
	}else{
		return nil,err
	}
}

func queryLabRun(c*gin.Context) (interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("create nest run invalid lab id or run id")
	}
	run, err := models.QueryRunDetail(runId)
	if err == nil && run.LabId != labId {
		return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid lab id passed for runs")
	}
	return run,err
}
func sysQueryLabRun(c*gin.Context)(interface{},APIError){
	_,runId := parseLabRunId(c)
	return models.QueryRunDetail(runId)
}

func getAllLabRuns(c*gin.Context) (interface{},APIError){
	labId,_ := parseLabRunId(c)
	if labId == 0 {
		return nil,exports.ParameterError("invalid lab id")
	}
	cond,err := checkSearchCond(c,nil)
	if err != nil {
		return nil,err
	}
	data,err := models.ListAllLabRuns(cond,labId)
	return makePagedQueryResult(cond,data,err)
}
func sysGetAllLabRuns(c*gin.Context)(interface{},APIError){
	labId,_ := parseLabRunId(c)
	cond,err := checkSearchCond(c,nil)
	if err != nil {
		return nil,err
	}
	data,err := models.ListAllLabRuns(cond,labId)
	return makePagedQueryResult(cond,data,err)
}

func queryLabRunStats(c*gin.Context) (interface{},APIError){
	labId,_ := parseLabRunId(c)
	return models.QueryLabStats(labId,c.Query("group"))
}

func sysQueryLabRunStats(c*gin.Context)(interface{},APIError){
    return nil,exports.NotImplementError()
}
func postLabRuns(c*gin.Context)(interface{},APIError) {
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("invalid lab id or run id")
	}
	req := &exports.CreateJobRequest{}
	switch(c.Query("action")){
	  case "open_visual":
	  	   if err := c.ShouldBindJSON(req);err != nil {
	  	   	  return nil,exports.ParameterError("invalid json data")
		   }
		   return openLabRunVisual(labId,runId,req)
	  case "close_visual":
	  	   return nil,models.TryKillNestRun(labId,runId,exports.AILAB_RUN_VISUALIZE)
	  case "save":
		  if err := c.ShouldBindJSON(req);err != nil {
			  return nil,exports.ParameterError("invalid json data")
		  }
		  return saveLabRun(labId,runId,req)
	  case "kill":
	  	  return nil,models.KillLabRun(labId,runId)
	  case "pause":
	  	  return nil,models.PauseLabRun(labId,runId)
	  case "resume":
	  	  return models.ResumeLabRun(labId,runId)
	  default:
	  	  return nil,exports.NotImplementError()
	}
}

func sysPostLabRuns(c*gin.Context)(interface{},APIError){
	_,runId := parseLabRunId(c)
	if len(runId) == 0 {
		return nil,exports.ParameterError("invalid run id")
	}
	switch(c.Query("action")){
	case "kill":
		return nil,models.KillLabRun(0,runId)
	case "pause":
		return nil,models.PauseLabRun(0,runId)
	case "resume":
		return models.ResumeLabRun(0,runId)
	default:
		return nil,exports.NotImplementError()
	}
}


func delLabRun(c*gin.Context) (interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("invalid lab id or run id")
	}
	return nil,models.TryDeleteLabRun(labId,runId)
}
func openLabRunVisual(labId uint64,runId string,req*exports.CreateJobRequest) (interface{},APIError){

	 req.JobFlags = exports.RUN_FLAGS_SINGLE_INSTANCE | exports.RUN_FLAGS_RESUMEABLE
	 req.JobType  = exports.AILAB_RUN_VISUALIZE
	 _, err := models.CreateLabRun(labId,runId,req)
	 if err == nil {// created new run
	 	return nil,exports.RaiseAPIError(exports.AILAB_WOULD_BLOCK,"wait to start visual job ...")
	 }else if err.Errno() == exports.AILAB_SINGLETON_RUN_EXISTS {// exists old job
	 	 _, err := models.ResumeLabRun(labId,err.Error())
	 	 if err == nil{//@todo: here should return visit url

		 }
		 return nil,err
	 }else{
	 	return nil,err
	 }
}


func sysCleanLabRuns(c*gin.Context) (interface{},APIError){
	 return nil,exports.NotImplementError()
}

func sysResetCleanStrategy(c*gin.Context) (interface{},APIError){
	 return nil,exports.NotImplementError()
}

func parseLabRunId(c*gin.Context) (labId uint64,runId string){
	value := c.Param("lab")
	var err error
	if labId,err = strconv.ParseUint(value, 0, 64);err != nil {
		labId = 0
	}
	runId = c.Param("runId")
	return
}

