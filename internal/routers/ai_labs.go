
package routers

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/gin-gonic/gin"
	"strconv"
)

func AddGroupAILab(r *gin.Engine){

	group := r.Group("/api/v1/labs")

	group.GET("/", wrapper(getAllLabs))
	group.GET("/:lab", wrapper(queryLab))
	group.PUT("/:lab", wrapper(updateLab))
	group.POST("/", wrapper(createLab))
	group.POST("/batch", wrapper(batchCreateLab))
	group.POST("/deletes",wrapper(batchDeleteLab))
	group.POST("/kills",wrapper(batchKillLab))
	group.POST("/clear",wrapper(batchClearLab))
	group.DELETE("/:lab", wrapper(deleteLab))
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
		 if err := c.ShouldBindJSON(labs);err != nil{
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
	  if len(lab.Group) == 0 || len(lab.Name) == 0{
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
	  if len(lab.Namespace) == 0{
	  	return exports.ParameterError("invalid namespace")
	  }
	  lab.CreatedAt = models.UnixTime{}
	  lab.UpdatedAt = models.UnixTime{}
	  lab.DeletedAt = nil
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
	  if len(req.Namespace) == 0{
			return nil,exports.ParameterError("batch labs invalid namespace")
	  }
	  for index,item:= range(req.Labs) {
	  	 labs = append(labs,models.Lab{
			 App:         req.App,
			 Group:       req.Group,
			 Creator:     req.Creator,
			 Namespace:   req.Namespace,
		 })

	  	 err  = func(lab*models.Lab)APIError{

	  	 	 if v ,ok := item["description"];ok {
				 lab.Description,_ = v.(string)
			 }
			 if v ,ok := item["name"];ok {
				 lab.Name ,_= v.(string)
			 }
			 if v ,ok := item["classify"];ok {
				 lab.Classify,_ = v.(string)
			 }
			 if v ,ok := item["type"];ok {
				 lab.Type,_ = v.(string)
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
