package models

import (
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/util/uuid"
)


type Run struct{
	 RunId   string `json:"runId" gorm:"primary_key"`
	 LabId   uint64 `json:"labId" gorm:"index;not null"`
	 Group   string `json:"group" gorm:"index;not null"`    // user defined group
	 Name    string `json:"name"`
	 Started uint64 `json:"starts"`
	 Num      uint64 `json:"num"`                  // version number among lab or parent run
	 JobType string `json:"jobType"`               // system defined job types
	 Parent  string `json:"parent" gorm:"index"`   // created by parent run
	 Creator string `json:"creator"`
	 CreatedAt UnixTime  `json:"createdAt"`
	 DeletedAt *UnixTime `json:"deletedAt,omitempty"`
	 Description string  `json:"description"`
	 Start      UnixTime `json:"start"`
	 End        UnixTime `json:"end"`
	 Status     int      `json:"status"`
	 Arch       string   `json:"arch"`
	 Cmd        string   `json:"cmd"`
	 Image      string   `json:"image"`
	 Tags      *JsonMetaData `json:"tags,omitempty"`
	 Config    *JsonMetaData `json:"config,omitempty"`
	 Resource  *JsonMetaData `json:"resource,omitempty"`
	 Envs      *JsonMetaData `json:"envs"`
	 Endpoints *JsonMetaData `json:"endpoints"`
	 Quota     *JsonMetaData `json:"quota,omitempty"`
	 Output     string       `json:"output"`
	 Progress   *JsonMetaData `json:"progress,omitempty"`
	 Result     *JsonMetaData `json:"result,omitempty"`
	 Flags      uint64        `json:"flags"`   // some system defined attributes this run behaves
}

const (
	list_runs_fields=""
)

type BasicMLRunContext struct{
	  BasicLabInfo
	  RunId   string
	  Status  int
	  JobType string
	  Output  string
	  //track how many times nest run started
	  Started  uint64
	  Flags    uint64
}

func (ctx*BasicMLRunContext) IsNestRun() bool{
     return len(ctx.RunId) > 0
}
func (ctx*BasicMLRunContext) IsLabRun() bool {
     return len(ctx.RunId) == 0
}
func (ctx*BasicMLRunContext) PrepareJobStatusChange() *JobStatusChange {
	 return &JobStatusChange{
		 RunId:    ctx.RunId,
		 JobType:  ctx.JobType,
		 Flags:    ctx.Flags,
		 Status:   ctx.Status,
	 }
}

func  newLabRun(mlrun * BasicMLRunContext,req*exports.CreateJobRequest) *Run{
	  run := &Run{
		  RunId:       string(uuid.NewUUID()),
		  LabId:       mlrun.ID,
		  Group:       req.JobGroup,
		  Name:        req.Name,
		  JobType:     req.JobType,
		  Parent:      mlrun.RunId,
		  Creator:     req.Creator,
		  Description: req.Description,
		  Arch:        req.Arch,
		  Cmd:         req.Cmd,
		  Image:       req.Engine,
		  Output:      req.OutputPath,
		  Flags:       req.JobFlags,
	  }
	  if (req.JobFlags & exports.RUN_FLAGS_SINGLE_INSTANCE) != 0 {
	  	 if mlrun.IsLabRun() {
	  	 	run.Num = mlrun.Starts
		 }else{
		 	run.Num = mlrun.Started
		 }
	  }
	  if len(req.Tags) > 0 {
	  	 run.Tags=&JsonMetaData{}
	  	 run.Tags.Save(req.Tags)
	  }
	  if len(req.Config) > 0{
	  	 run.Config=&JsonMetaData{}
	  	 run.Config.Save(req.Config)
	  }
	  if len(req.Resource) > 0{
		 run.Resource=&JsonMetaData{}
	 	 run.Resource.Save(req.Resource)
	  }
	  if len(req.Envs) > 0{
			run.Envs=&JsonMetaData{}
			run.Envs.Save(req.Envs)
	  }
	  if len(req.Endpoints) > 0{
			run.Endpoints=&JsonMetaData{}
			run.Endpoints.Save(req.Endpoints)
	  }
	  if true {
		run.Quota=&JsonMetaData{}
		run.Quota.Save(req.Quota)
	  }
	  return run
}

func  CreateLabRun(labId uint64,runId string,req*exports.CreateJobRequest) (run*Run,err APIError){

	err = execDBTransaction(func(tx *gorm.DB) APIError {

		mlrun ,err := getBasicMLRunInfo(tx,labId,runId)
		if err == nil {
			err = preCheckCreateRun(tx,mlrun,req)
		}
		if err == nil{
			run  = newLabRun(mlrun,req)
			err  = allocateLabRunStg(run,mlrun)
			if err == nil{
				err  = wrapDBUpdateError(tx.Create(&run),1)
			}
		}
		if err == nil{
            err = completeCreateRun(tx,mlrun,req)
		}
		if err == nil {
			err = doLogStartRun(tx,run.RunId)
		}
		return err
	})
	return
}
func  QueryRunDetail(runId string) (*Run,APIError){
	run := &Run{}
	err := wrapDBQueryError(db.First(run,"id=?",runId))
	return run,err
}

func  scanNestChildRuns(tx*gorm.DB,runId string,childs []JobStatusChange) ([]JobStatusChange,APIError){
	var result []JobStatusChange
	err := wrapDBQueryError(tx.Find(&result,"parent=?",runId))
	copy(childs,result)
	return childs,err
}

func tryKillRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) APIError{
    return exports.NotImplementError()
}
func tryResumeRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext)APIError{
	return exports.NotImplementError()
}
func tryDeleteRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext)APIError{
	return exports.NotImplementError()
}
func  tryRecursiveCtrlRun(tx*gorm.DB, mlrun*BasicMLRunContext,jobType string,deepScan bool ,
	      executor func(*gorm.DB,*JobStatusChange,*BasicMLRunContext) APIError) APIError{

	    inst := tx.Table("runs").Select("run_id,status,flags,job_type")
	    if len(jobType) > 0 {
	    	inst = inst.Where("job_type=?",jobType)
		}
		
		result,err := scanNestChildRuns(inst,mlrun.RunId,[]JobStatusChange{})
		if err == nil && deepScan {

			for i:=0 ;i<len(result);i++ {
				result,err = scanNestChildRuns(inst,result[i].RunId,result)
				if err != nil {
					return err
				}
			}
		}
		for _,item := range(result) {
			if err = executor(tx,&item,mlrun);err != nil {
				return err
			}
		}
		// update lab statistics
		return mlrun.Save(tx)
}

func  TryKillNestRun(labId uint64,runId string , jobType string) APIError{
	return execDBTransaction(func(tx *gorm.DB) APIError {

		  assertCheck(len(runId)>0,"TryKillNestRun must have parent runId")

		  mlrun ,err := getBasicMLRunInfo(tx,labId,runId)
		  if err == nil{
			  err = tryRecursiveCtrlRun(tx,mlrun,jobType,false,tryKillRun)
		  }
		  return err
	})
}

func TryDeleteLabRun(labId uint64,runId string) APIError{
	return execDBTransaction(func(tx *gorm.DB) APIError {

		assertCheck(len(runId)>0,"TryDeleteLabRun must have parent runId")

		mlrun ,err := getBasicMLRunInfo(tx,labId,runId)
		if err == nil{
			err = tryRecursiveCtrlRun(tx,mlrun,"",true,tryDeleteRun)
		}
		return err
	})

}

func KillLabRun(labId uint64,runId string) APIError {
	return execDBTransaction(func(tx*gorm.DB)APIError{
		mlrun,err := getBasicMLRunInfo(tx,labId,runId)
		if err == nil {
			err = tryKillRun(tx,mlrun.PrepareJobStatusChange(),mlrun)
		}
		if err == nil {
			err = mlrun.Save(tx)
		}
		return err
	})
}

func ResumeLabRun(labId uint64,runId string) (mlrun *BasicMLRunContext,err APIError){
	err= execDBTransaction(func(tx*gorm.DB)APIError{
		mlrun,err = getBasicMLRunInfo(tx,labId,runId)
		if err == nil {
			err = tryResumeRun(tx,mlrun.PrepareJobStatusChange(),mlrun)
		}
		if err == nil {
			err = mlrun.Save(tx)
		}
		return err
	})
	return
}

func PauseLabRun(labId uint64,runId string) APIError{
	 return exports.NotImplementError()
}
func ListAllLabRuns(req*exports.SearchCond,labId uint64) (interface{} ,APIError){
	inst := db.Select(list_runs_fields)
	if labId != 0 {
		inst = inst.Where("id=?",labId)
	}
	return makePagedQuery(inst,req, &[]Run{})
}
func  getBasicMLRunInfo(tx*gorm.DB,labId uint64,runId string) (mlrun*BasicMLRunContext,err APIError){

	  mlrun = &BasicMLRunContext{}
      if len(runId) == 0 {
		  lab,err := getBasicLabInfo(tx,labId)
		  if err != nil{
		  	return nil,err
		  }
		  mlrun.BasicLabInfo=*lab
	  }else{
		  err = checkDBQueryError(db.Table("runs").Set("gorm:query_option","FOR UPDATE").
			  Select("id,starts,location,statistics, run_id,status,job_type,output,started,flags").
			  Joins("left join labs on runs.lab_id=labs.id").
			  Where("run_id=?",runId).Row().Scan(mlrun))

		  if err == nil && labId != 0 && labId != mlrun.ID {
			  err = exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid lab id passed for runs")
		  }
	  }
	  return
}


func tryDeleteLabRuns(tx*gorm.DB,labId uint64) APIError{
	 runId := ""

	 err := checkDBQueryError(tx.Model(&Run{}).Select("run_id").Limit(1).
	 	Where("lab_id=? and status < ?",labId,exports.RUN_STATUS_FAILED).Row().Scan(&runId))
	 if err == nil {
	 	return exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE,"still have runs active !")
	 }else if err.Errno() == exports.AILAB_NOT_FOUND{
	 	return wrapDBUpdateError(tx.Delete(&Run{},"lab_id=?",labId),0)

	 }else{
	 	return err
	 }
}
func tryDeleteLabRunsByGroup(tx*gorm.DB, labs []uint64) APIError{
	runId := ""

	err := checkDBQueryError(tx.Model(&Run{}).Select("run_id").Limit(1).
		Where("lab_id in ? and status < ?",labs,exports.RUN_STATUS_FAILED).Row().Scan(&runId))
	if err == nil {
		return exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE,"still have runs active !")
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		return wrapDBUpdateError(tx.Delete(&Run{},"lab_id in ?",labs),0)
	}else{
		return err
	}
}

func preCheckCreateRun(tx*gorm.DB, mlrun*BasicMLRunContext,req*exports.CreateJobRequest) APIError {

	if mlrun.IsLabRun()  {//create lab run
		if (req.JobFlags & exports.RUN_FLAGS_SINGLE_INSTANCE) != 0{
			exists := ""
			err := checkDBQueryError(tx.Table("runs").Select("run_id").
				Where("lab_id=? and job_type=?",mlrun.ID,req.JobType).
				Row().Scan(&exists))
			if err == nil{//exists singleton instance
				return exports.RaiseAPIError(exports.AILAB_SINGLETON_RUN_EXISTS,exists)
			}else if err.Errno() != exports.AILAB_NOT_FOUND {
				return err
			}
		}else{//track lab run starts
			mlrun.Starts ++
		}
	}else{// create nest run
		if (req.JobFlags & exports.RUN_FLAGS_SINGLE_INSTANCE) != 0{
			exists := ""
			err    := checkDBQueryError(tx.Table("runs").Select("run_id").
				Where("parent=? and job_type=?",mlrun.RunId,req.JobType).
				Row().Scan(&exists))
			if err == nil{//exists singleton instance
				return exports.RaiseAPIError(exports.AILAB_SINGLETON_RUN_EXISTS,exists)
			}else if err.Errno() != exports.AILAB_NOT_FOUND {
				return err
			}
		}else{//track run nested starts
			mlrun.Started++
		}
	}
	return nil
}

func completeCreateRun(tx*gorm.DB, mlrun*BasicMLRunContext,req*exports.CreateJobRequest) (err APIError){
	 if (req.JobFlags & exports.RUN_FLAGS_SINGLE_INSTANCE) == 0 {
		 if mlrun.IsLabRun() {//create lab run
	        err = wrapDBUpdateError(tx.Table("labs").Where("id=?",mlrun.ID).Update("starts",mlrun.Starts),1)
		 }else{// create nest run
		 	err = wrapDBUpdateError(tx.Table("runs").Where("run_id=?",mlrun.RunId).Update("started",mlrun.Started),1)
		 }
	 }
	 if err == nil{
	 	mlrun.JobStatusChange(&JobStatusChange{
			JobType:  req.JobType,
			Status:   -1,
			StatusTo: exports.RUN_STATUS_INIT,
		})
	 	err = mlrun.Save(tx)
	 }
	 return err
}



