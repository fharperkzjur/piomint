package routers

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/internal/services"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func AddGroupTraining(r*gin.Engine){

	group := r.Group("/api/v1/labs/:lab")

	group.POST("/runs", wrapper(submitLabRun))
	group.POST("/runs/:runId/evaluates", wrapper(submitLabEvaluate))
    // operates on lab runs : open_visual/close_visual  , pause|resume|kill|stop
	group.POST("/runs/:runId", wrapper(postLabRuns))
    // support train&evaluate job list
	group.GET("/runs", wrapper(getAllLabRuns))
	group.GET("/runs/:runId",wrapper(queryLabRun))
	group.GET("/stats",wrapper(queryLabRunStats))

	group.GET( "/runs/:runId/files",   wrapper(listLabRunFiles))
	group.GET("/runs/:runId/view",     viewLabRunFiles)

	group.DELETE("/runs/:runId", wrapper(delLabRun))
    // following interfce should only be called by admin role users
	group = r.Group("/api/v1/runs")
	group.GET("",wrapper(sysGetAllLabRuns))
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
	 return services.ReqCreateRun(labId,"",req,false,false)
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
	return services.ReqCreateRun(labId,runId,req,false,false)
}

func saveLabRun(labId uint64, runId string,req *exports.CreateJobRequest) (interface{},APIError){
	req.JobType  = exports.AILAB_RUN_SAVE
	req.JobFlags = exports.AILAB_RUN_FLAGS_SINGLE_INSTANCE | exports.AILAB_RUN_FLAGS_AUTO_DELETED
	run, err := services.ReqCreateRun(labId,runId,req,true,false)
	if err == nil {// created new run
		return nil,exports.RaiseAPIError(exports.AILAB_WOULD_BLOCK,"wait to start save job ...")
	}else if err.Errno() == exports.AILAB_STILL_ACTIVE {// exists old job
		return nil,exports.RaiseAPIError(exports.AILAB_SERVER_BUSY, "old save job still active ...")
	}else{
		return run,err
	}
}

func queryLabRun(c*gin.Context) (interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("create nest run invalid lab id or run id")
	}
	run, err := models.QueryRunDetail(runId,false,-1)
	if err == nil && run.LabId != labId {
		return nil,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid lab id passed for runs")
	}
	return run,err
}
func sysQueryLabRun(c*gin.Context)(interface{},APIError){
	_,runId := parseLabRunId(c)
	return models.QueryRunDetail(runId,true,-1)
}

func getAllLabRuns(c*gin.Context) (interface{},APIError){
	labId,_ := parseLabRunId(c)
	if labId == 0 {
		return nil,exports.ParameterError("invalid lab id")
	}
	cond,err := checkSearchCond(c,exports.QueryFilterMap{
		"jobType":"job_type",
		"parent" :"parent",
		"status" : "status",
	})
	if err != nil {
		return nil,err
	}
	data,err := models.ListAllLabRuns(cond,labId,true)
	return makePagedQueryResult(cond,data,err)
}
func sysGetAllLabRuns(c*gin.Context)(interface{},APIError){
	cond,err := checkSearchCond(c,exports.QueryFilterMap{
		"jobType":"job_type",
		"parent" :"parent",
		"status" : "status",
	})
	if err != nil {
		return nil,err
	}
	data,err := models.ListAllLabRuns(cond,0,true)
	return makePagedQueryResult(cond,data,err)
}

func queryLabRunStats(c*gin.Context) (interface{},APIError){
	labId,_ := parseLabRunId(c)
	return models.QueryLabStats(labId,c.Query("group"))
}

func sysQueryLabRunStats(c*gin.Context)(interface{},APIError){
    return models.QueryLabStats(0,c.Query("group"))
}
func postLabRuns(c*gin.Context)(interface{},APIError) {
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("invalid lab id or run id")
	}
	req := &exports.CreateJobRequest{}
	action:=c.Query("action")
	switch(c.Query("action")){
	  case "open_visual":
	  	   if err := c.ShouldBindJSON(req);err != nil {
	  	   	  return nil,exports.ParameterError("invalid json data")
		   }
		   return openLabRunVisual(labId,runId,req)
	  case "close_visual":
	  	   return models.KillNestRun(labId,runId,exports.AILAB_RUN_VISUALIZE,false)
	  case "save":
		  if err := c.ShouldBindJSON(req);err != nil {
			  return nil,exports.ParameterError("invalid json data")
		  }
		  return saveLabRun(labId,runId,req)
	  case "cancel_save":
	  	   return models.KillNestRun(labId,runId,exports.AILAB_RUN_SAVE,false)
	  case "kill":
	  	  return  models.KillLabRun(labId,runId,false)
	  case "pause":
	  	  return nil,models.PauseLabRun(labId,runId)
	  case "resume":
	  	  return models.ResumeLabRun(labId,runId)
	  case "clean":
	  	  return models.CleanLabRun(labId,runId)
	  default:
	  	  return nil,exports.NotImplementError("unsupport mlrun action:"+action)
	}
}

func sysPostLabRuns(c*gin.Context)(interface{},APIError){
	_,runId := parseLabRunId(c)
	if len(runId) == 0 {
		return nil,exports.ParameterError("invalid run id")
	}
	action := c.Query("action")
	switch(c.Query("action")){
	case "kill":
		return models.KillLabRun(0,runId,false)
	case "pause":
		return nil,models.PauseLabRun(0,runId)
	case "resume":
		return models.ResumeLabRun(0,runId)
	case "clean":
		return models.CleanLabRun(0,runId)
	default:
		return nil,exports.NotImplementError("unsupport sys mlrun action:"+action)
	}
}

func delLabRun(c*gin.Context) (interface{},APIError){
	labId,runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0 {
		return nil,exports.ParameterError("invalid lab id or run id")
	}
	return models.DeleteLabRun(labId,runId,false)
}

func openLabRunVisual(labId uint64,runId string,req*exports.CreateJobRequest) (interface{},APIError){

	 req.JobFlags = exports.AILAB_RUN_FLAGS_SINGLE_INSTANCE | exports.AILAB_RUN_FLAGS_RESUMEABLE
	 req.JobType  = exports.AILAB_RUN_VISUALIZE
	 run, err := services.ReqCreateRun(labId,runId,req,true,false)
	 if err == nil {// created new run
	 	return nil,exports.RaiseAPIError(exports.AILAB_WOULD_BLOCK,"wait to start visual job ...")
	 }else if err.Errno() == exports.AILAB_SINGLETON_RUN_EXISTS {// exists old job
	 	 job := run.(*models.JobStatusChange)
	 	 run, err := models.ResumeLabRun(labId,job.RunId)
	 	 if err == nil{
	 	 	return services.GetEndpointUrl(run)
		 }
		 return nil,err
	 }else{
	 	return nil,err
	 }
}


func sysCleanLabRuns(c*gin.Context) (interface{},APIError){
	 req := &ReqTargetLab{}
	 if err := c.ShouldBindJSON(req);err != nil {
		return nil,exports.ParameterError("batch clean lab invalid json data")
	 }
	 return models.CleanLabRunByGroup(req.Group,req.LabID)
}

func sysResetCleanStrategy(c*gin.Context) (interface{},APIError){
	 return nil,exports.NotImplementError("sysResetCleanStrategy")
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

func listLabRunFiles(c*gin.Context)(interface{},APIError){
	labId, runId := parseLabRunId(c)
	if labId == 0 || len(runId) == 0{
		return nil,exports.ParameterError("list run files invald labId or runId")
	}
	return services.ListRunFiles(labId,runId,c.Query("prefix"))
}

func viewLabRunFiles(c*gin.Context){

		labId, runId := parseLabRunId(c)
		if labId == 0 || len(runId) == 0{
			c.JSON(http.StatusBadRequest,exports.CommResponse{
				Code: exports.AILAB_PARAM_ERROR,
				Msg:  "list run files invald labId or runId",
			})
			return
		}
		if err := services.ServeFile(labId,runId,c.Query("prefix"),c);err != nil {
			c.JSON(http.StatusBadRequest,exports.CommResponse{
				Code: err.Errno(),
				Msg:  err.Error(),
			})
		}
}


