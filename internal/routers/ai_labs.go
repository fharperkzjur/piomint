
package routers

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/gin-gonic/gin"
	"strconv"
)

func AddGroupAILab(r *gin.Engine){

	rg := r.Group( exports.AILAB_API_VERSION +  "/labs")

	group := (*IAMRouteGroup)(rg)

	group.GET("", wrapper(getAllLabs),   "lab:list")
	group.GET("/:lab", wrapper(queryLab),"lab:view")
	group.PUT("/:lab", wrapper(updateLab),"lab:update")
	group.POST("", wrapper(createLab),    "lab:create")
	group.POST("/batch", wrapper(batchCreateLab),"labs:create")
	group.POST("/deletes",wrapper(batchDeleteLab),"labs:delete")
	group.POST("/kills",wrapper(batchKillLab), "labs:kill")
	group.POST("/clear",wrapper(batchClearLab),"labs:clear")
	group.DELETE("/:lab", wrapper(deleteLab),  "lab:delete")

}

func getAllLabs(c*gin.Context)(interface{},APIError){

	 cond,err := checkSearchCond(c,nil)
	 if err != nil {
	 	return nil,err
	 }
	 data,err := models.ListAllLabs(cond)
	 return makePagedQueryResult(cond,data,err)
}

func queryLab(c*gin.Context)(interface{},APIError){
	 if labId,err := strconv.ParseUint(c.Param("lab"),0,64);err == nil && labId > 0{
	 	return models.QueryLabDetail(labId)
	 }else{
	 	return nil,exports.ParameterError("invalid lab ID")
	 }
}

func updateLab(c*gin.Context)(interface{},APIError){
	 if labId,err := strconv.ParseUint(c.Param("lab"),0,64);err == nil && labId > 0{
		 labs := make(exports.RequestObject)
		 if err := c.ShouldBindJSON(&labs);err != nil{
			 return nil,exports.ParameterError("invalid update lab information")
		 }
		 return nil,models.UpdateLabInfo(labId,labs)
	 }else{
		 return nil,exports.ParameterError("invalid lab ID")
	 }
}

func createLab(c*gin.Context)(interface{},APIError){
	 lab := &models.Lab{}
	 if err := c.ShouldBindJSON(lab);err != nil{
	 	return nil,exports.ParameterError("invalid create lab information")
	 }
	 if err := checkCreateLab(lab);err != nil{
	 	return nil,err
	 }
	 err := models.CreateLab(lab)
	 return lab.ID,err
}

func batchCreateLab(c*gin.Context) (interface{},APIError){
	 req := &exports.ReqBatchCreateLab{}
	 if err := c.ShouldBindJSON(req);err != nil{
	 	return nil,exports.ParameterError("invalid batch create lab informations")
	 }
	 if labs,err := checkBatchCreateLab(req);err != nil{
	 	return nil,err
	 }else{
	 	err = models.BatchCreateLab(labs)
	 	return nil,err
	 }
}
type ReqTargetLab struct{
	Group string `json:"group"`
	LabID uint64 `json:"labId"`
}
func batchDeleteLab(c*gin.Context) (interface{},APIError){

	 req := &ReqTargetLab{}
	 if err := c.ShouldBindJSON(req);err != nil {
	 	return nil,exports.ParameterError("batch delete lab invalid json data")
	 }
	 return models.DeleteLabByGroup(req.Group,req.LabID)
}

func batchKillLab(c*gin.Context) (interface{},APIError){
	  req := &ReqTargetLab{}
	  if err := c.ShouldBindJSON(req);err != nil {
			return nil,exports.ParameterError("batch kill lab invalid json data")
	  }
	  return models.KillLabByGroup(req.Group,req.LabID)
}

func batchClearLab(c*gin.Context) (interface{},APIError){

	req := &ReqTargetLab{}
	if err := c.ShouldBindJSON(req);err != nil {
		return nil,exports.ParameterError("batch delete lab invalid json data")
	}
	return models.ClearLabByGroup(req.Group,req.LabID)
}

func deleteLab(c*gin.Context) (interface{},APIError){
	if labId,err := strconv.ParseUint(c.Param("lab"),0,64);err == nil && labId > 0{
        return nil,models.DeleteLab(labId)
	}else{
		return nil,exports.ParameterError("invalid delete lab ID")
	}
}

func checkCreateLab(lab*models.Lab) APIError{
	  lab.ID=0
	  if len(lab.Bind) == 0 || len(lab.Name) == 0{
	  	return exports.ParameterError("lab group name cannot be empty")
	  }
	  if len(lab.App) == 0 {
	  	return exports.ParameterError("invalid app name")
	  }
	  if len(lab.Type) == 0 {
	  	return exports.ParameterError("invalid lab type enum")
	  }
	  if len(lab.Creator) == 0 {
		return exports.ParameterError("invalid creator")
	  }
	  if lab.UserId == 0 {
	  	return exports.ParameterError("invalid user id")
	  }
	  if len(lab.Namespace) == 0{
	  	return exports.ParameterError("invalid namespace")
	  }
	  lab.CreatedAt = models.UnixTime{}
	  lab.UpdatedAt = models.UnixTime{}
	  lab.DeletedAt = 0
	  lab.Starts = 0
	  lab.Statistics = nil
	  return nil
}
func checkBatchCreateLab(req*exports.ReqBatchCreateLab)(labs []models.Lab,err APIError){

	  if len(req.Labs) == 0{
	  	return nil,exports.ParameterError("batch labs cannot be empty")
	  }
	  if len(req.Group) == 0 {
			return nil,exports.ParameterError("batch labs group cannot be empty")
	  }
	  if len(req.App) == 0 {
			return nil,exports.ParameterError("batch labs invalid app name")
	  }
	  if len(req.Creator) == 0 {
			return nil,exports.ParameterError("batch labs invalid creator")
	  }
	  if req.UserId == 0 {
	  	    return nil,exports.ParameterError("batch labs invalid user id")
	  }
	  if len(req.Namespace) == 0{
			return nil,exports.ParameterError("batch labs invalid namespace")
	  }
	  for index,item:= range(req.Labs) {
	  	 labs = append(labs,models.Lab{
			 App:         req.App,
			 Bind:        req.Group,
			 Creator:     req.Creator,
			 UserId:      req.UserId,
			 Namespace:   req.Namespace,
			 ProjectName: req.ProjectName,
			 //@add: retrieve org & group information from jwt context
			 OrgId:       req.OrgId,
			 OrgName:     req.OrgName,
			 UserGroupId: req.UserGroupId,
		 })

	  	 err  = func(lab*models.Lab)APIError{
	  	 	 lab.Description = item.Description
	  	 	 lab.Name = item.Name
	  	 	 lab.Classify = item.Classify
	  	 	 lab.Type = item.Type
	  	 	 if len(item.Tags) > 0 {
	  	 	 	lab.Tags = &models.JsonMetaData{}
	  	 	 	lab.Tags.Save(item.Tags)
			 }
			 if len(item.Meta) > 0 {
				 lab.Meta = &models.JsonMetaData{}
				 lab.Meta.Save(item.Meta)
			 }
			 if len(lab.Name) == 0 {
			 	return exports.ParameterError("invalid lab name")
			 }
			 if len(lab.Type) == 0 {
				 return exports.ParameterError("invalid lab type enum")
			 }
			 return nil
		 }(&labs[index])

	  	 if err != nil{
	  	 	break
		 }
	  }
	  return
}
