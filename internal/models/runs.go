package models

import (
	"database/sql"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/util/uuid"
	"time"
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
	select_run_status_change = "run_id,status,flags,job_type"
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
func (ctx*BasicMLRunContext) HasCompleteOK() bool {
	 return ctx.Status == exports.RUN_STATUS_SUCCESS
}
func (ctx*BasicMLRunContext) NeedClean() bool {
	 return ctx.Status >= exports.RUN_STATUS_PRE_CLEAN
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

func  CreateLabRun(labId uint64,runId string,req*exports.CreateJobRequest,enableRepace bool,syncPrepare bool) (result interface{},err APIError){

	err = execDBTransaction(func(tx *gorm.DB) APIError {

		mlrun ,err := getBasicMLRunInfo(tx,labId,runId)
		if err == nil {
			old, err := preCheckCreateRun(tx,mlrun,req)
			if err != nil && err.Errno() == exports.AILAB_SINGLETON_RUN_EXISTS {
				if !enableRepace {
					result = old
					return err
				}
				//discard old run if possible
				_,err = tryForceDeleteRun(tx,old,mlrun)
			}
		}
		var run * Run
		if err == nil{
			run  = newLabRun(mlrun,req)
			err  = allocateLabRunStg(run,mlrun)
			if err == nil{
				//@mark: no resource request !!!
				if run.Resource == nil || run.Resource.Empty(){
                    run.Flags |= exports.RUN_FLAGS_PREPARE_SUCCESS
                    run.Status = exports.RUN_STATUS_STARTING
				}else if syncPrepare {//@mark: synchronoulsy prepare create deleted run first , then change to starting when prepare success
					run.DeletedAt = &UnixTime{time.Now()}
				}
				err  = wrapDBUpdateError(tx.Create(&run),1)
			}
		}
		if err == nil{
            err = completeCreateRun(tx,mlrun,req,run)
		}
		if err == nil{
			result = run
		}
		return err
	})
	return
}
func  QueryRunDetail(runId string,unscoped bool,status int) (run*Run,err APIError){
	run  = &Run{}
	inst := db
	if unscoped {
		inst = inst.Unscoped()
	}
	if status >= 0 {
		inst = inst.Where("status=?",status)
	}
	err =  wrapDBQueryError(inst.First(run,"run_id=?",runId))
	return
}
func SelectAnyLabRun(labId uint64) (run*Run,err APIError){
	run = &Run{}
	return run,wrapDBQueryError(db.Unscoped().First(run,"lab_id=?",labId))
}

func tryResumeRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError){
	if run.IsStopping() {
		return 0,exports.RaiseAPIError(exports.AILAB_INVALID_RUN_STATUS)
	}else if run.RunActive() {//already running or started
		return 0,nil
	}else if !run.EnableResume(){
		return 0,exports.RaiseAPIError(exports.AILAB_RUN_CANNOT_RESTART)
	}

	if run.HasInitOK() {
		run.StatusTo =  exports.RUN_STATUS_STARTING
	}else{
		run.StatusTo = exports.RUN_STATUS_INIT
	}

	err := wrapDBUpdateError(tx.Table("runs").Update("status",run.StatusTo).
		Where("run_id=?",run.RunId),1)
	if err == nil {
		mlrun.JobStatusChange(run)
		err = doLogStartingRun(tx,run.RunId,run.StatusTo)
	}
	return 1,err
}

func tryKillRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError){
	if !run.RunActive() {// no need to kill
		return 0,nil
	}
	if run.StatusTo >= exports.RUN_STATUS_FAILED {//user defined end status

	}else if run.IsIniting() {// kill to abort immediatley
		run.StatusTo = exports.RUN_STATUS_ABORT
	}else{
		run.StatusTo = exports.RUN_STATUS_KILLING
	}
	err := wrapDBUpdateError(tx.Table("runs").Update("status",run.StatusTo).
		Where("run_id=?",run.RunId),1)
	if err == nil {
		mlrun.JobStatusChange(run)
		if run.StatusTo == exports.RUN_STATUS_KILLING {
			err = doLogKillRun(tx,run.RunId)
		}
	}
	return 1,err
}

func tryDeleteRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError){
	if  run.RunActive() {//cannot delete a active run
		return 0,exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE)
	}
	run.StatusTo = -1
	err := wrapDBUpdateError(tx.Delete(&Run{},"run_id=?",run.RunId),1)
	if err == nil {
		mlrun.JobStatusChange(run)
	}
	return 1,err

}
func tryForceDeleteRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext)(uint64,APIError){
	if  run.RunActive() {//cannot delete a active run
		return 0,exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE)
	}
	run.StatusTo = -1
	err := wrapDBUpdateError(tx.Where("run_id=?",run.RunId).Updates(
		    map[string]interface{}{
				"deleted_at": UnixTime{time.Now()},
				"status":     exports.RUN_STATUS_PRE_CLEAN,
			}),1)
	if err == nil {
		mlrun.JobStatusChange(run)
		err = doLogCleanRun(tx,run.RunId)
	}
	return 1,err
}

func tryCleanRunWithDeleted(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError) {
		if  run.RunActive() {//cannot delete a active run , should never happen
			return 0,exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE)
		}
	    err := wrapDBUpdateError(tx.Table("runs").UpdateColumn("status",exports.RUN_STATUS_PRE_CLEAN).
		    Where("run_id=?",run.RunId),1)
	    if err == nil {
	    	err = doLogCleanRun(tx,run.RunId)
	  }
	  return 1,err
}


func  tryRecursiveOpRuns(tx*gorm.DB, mlrun*BasicMLRunContext,jobType string,deepScan bool ,applySelf bool,
	      executor func(*gorm.DB,*JobStatusChange,*BasicMLRunContext) (uint64,APIError)) (counts uint64,err APIError){

	    inst := tx.Table("runs").Select(select_run_status_change)
	    if len(jobType) > 0 {
	    	inst = inst.Where("job_type=?",jobType)
		}
		result := []JobStatusChange{}
		if mlrun.IsLabRun() {
            err = wrapDBQueryError(inst.Find(&result,"lab_id=?",mlrun.ID))
		}else if applySelf {
			result = append(result,*mlrun.PrepareJobStatusChange())
		}else{
			err = wrapDBQueryError(inst.Find(&result,"parent=?",mlrun.RunId))
		}
		if deepScan {
			for i:=0; err != nil && i<len(result);i++ {
				err = execDBQuerRows(inst.Where("parent=?",result[i].RunId),func(rows*sql.Rows)APIError{

					job := &JobStatusChange{}
					err := checkDBScanError(rows.Scan(job))
					result = append(result,*job)
                    return err

				})
			}
		}
	    cnt := uint64(0)
		for i:=0; err != nil && i<len(result);i++{
			cnt,err = executor(tx,&result[i],mlrun)
			counts += cnt
		}
        // update lab statistics
        if err == nil {
			err = mlrun.Save(tx)
		}
		return
}

func  KillNestRun(labId uint64,runId string , jobType string, deepScan bool ) (counts uint64,err APIError){
	err = execDBTransaction(func(tx *gorm.DB) APIError {
		  assertCheck(len(runId)>0,"KillNestRun must have runId")
		  mlrun ,err := getBasicMLRunInfo(tx,labId,runId)
		  if err == nil{
			  counts,err = tryRecursiveOpRuns(tx,mlrun,jobType,deepScan,false,tryKillRun)
		  }
		  return err
	})
	return
}
func KillLabRun(labId uint64,runId string,deepScan bool) (counts uint64,err APIError) {
	err =  execDBTransaction(func(tx*gorm.DB)APIError{
		assertCheck(len(runId)>0,"KillLabRun must have runId")
		mlrun,err := getBasicMLRunInfo(tx,labId,runId)
		if err == nil {
			counts,err = tryRecursiveOpRuns(tx,mlrun,"",deepScan,true,tryKillRun)
		}
		return err
	})
	return
}
func DeleteLabRun(labId uint64,runId string,force bool) (counts uint64, err APIError){
	err = execDBTransaction(func(tx *gorm.DB) APIError {

		assertCheck(len(runId)>0,"DeleteLabRun must have runId")

		mlrun ,err := getBasicMLRunInfo(tx,labId,runId)
		if err == nil{
			if !force {
				counts, err = tryRecursiveOpRuns(tx,mlrun,"",true,true,tryDeleteRun)
			}else{
				counts, err = tryRecursiveOpRuns(tx,mlrun,"",true,true,tryForceDeleteRun)
			}
		}
		return err
	})
	return
}
func CleanLabRun(labId uint64,runId string) (counts uint64,err APIError) {
	err =  execDBTransaction(func(tx*gorm.DB)APIError{
		assertCheck(len(runId)>0,"CleanLabRun must have runId")
		//@mark:  here only need to traverse all deleted runs !!!
		tx = tx.Unscoped().Where("deleted_at is not null")
		mlrun,err := getBasicMLRunInfo(tx,labId,runId)
		if err == nil {
			counts,err = tryRecursiveOpRuns(tx,mlrun,"",true,true,tryCleanRunWithDeleted)
		}
		return err
	})
	return
}

func ResumeLabRun(labId uint64,runId string) (mlrun *BasicMLRunContext,err APIError){

	err = execDBTransaction(func(tx *gorm.DB) APIError {
		assertCheck(len(runId)>0,"ResumeLabRun must have runId")
		mlrun ,err := getBasicMLRunInfo(tx,labId,runId)
		if err == nil{
			_, err = tryRecursiveOpRuns(tx,mlrun,"",false,true,tryResumeRun)
		}
		return err
	})
	return
}

func PauseLabRun(labId uint64,runId string) APIError{
	 return exports.NotImplementError("PauseLabRun")
}
func ListAllLabRuns(req*exports.SearchCond,labId uint64) (interface{} ,APIError){
	inst := db.Select(list_runs_fields)
	if labId != 0 {
		inst = inst.Where("id=?",labId)
	}
	return makePagedQuery(inst,req, &[]Run{})
}

func  PrepareRunSuccess(runId string,resource* JsonMetaData,isRollback bool) APIError{
	if isRollback {// process deleted RUN_STATUS_INIT
		return execDBTransaction(func(tx*gorm.DB)APIError{

			mlrun ,err := getBasicMLRunInfo(tx.Unscoped().Where("runs.deleted_at is not null and runs.status=?",exports.RUN_STATUS_INIT),0 ,runId)
			if err != nil{
				if err.Errno() == exports.AILAB_NOT_FOUND {// context not exists
					return nil
				}else{
					return err
				}
			}
			err = wrapDBUpdateError( tx.Table("runs").Where("run_id=?",runId).
				UpdateColumns(map[string]interface{}{
					"status" : exports.RUN_STATUS_STARTING,
					"deleted_at":nil,
					"flags": gorm.Expr("flags|?",exports.RUN_FLAGS_PREPARE_SUCCESS),
					"resource":resource,}) ,1)
			if err == nil {
				jobs := mlrun.PrepareJobStatusChange()
				jobs.Status = -1
				jobs.StatusTo = exports.RUN_STATUS_STARTING
				mlrun.JobStatusChange(jobs)
				err = mlrun.Save(tx)
			}
			if err == nil {
				err = doLogStartingRun(tx,runId,exports.RUN_STATUS_STARTING)
			}
			return err
		})

	}else{// process normal RUN_STATUS_INIT
		return execDBTransaction(func(tx*gorm.DB)APIError{

			err := wrapDBUpdateError( tx.Table("runs").Where("run_id=? and status=?",runId,exports.RUN_STATUS_INIT).
				Updates(map[string]interface{}{
					"status" : exports.RUN_STATUS_STARTING,
					"flags": gorm.Expr("flags|?",exports.RUN_FLAGS_PREPARE_SUCCESS),
					"resource":resource,}) ,1)
			if err != nil && err.Errno() == exports.AILAB_DB_UPDATE_UNEXPECT {// context not exists
				return nil
			}
			if err == nil {
				err = doLogStartingRun(tx,runId,exports.RUN_STATUS_STARTING)
			}
			return err
		})
	}
}

func PrepareRunFailed(runId string,isRollback bool) APIError {
	if isRollback {
		return execDBTransaction(func(tx*gorm.DB)APIError{

			mlrun,err := getBasicMLRunInfo(tx.Unscoped().Where("deleted_at is not null and status=?",exports.RUN_STATUS_INIT),0,runId)
			if err != nil{
				if err.Errno() == exports.AILAB_NOT_FOUND {// context not exists
					return nil
				}else{
					return err
				}
			}
			jobs := mlrun.PrepareJobStatusChange()
			jobs.Status  = exports.RUN_STATUS_FAILED
			_,err = tryCleanRunWithDeleted(tx,jobs,mlrun)
			return err
		})
	}else{
		return execDBTransaction(func(tx*gorm.DB)APIError{

			mlrun,err := getBasicMLRunInfo(tx.Where("status=?",exports.RUN_STATUS_INIT),0,runId)
			if err != nil{
				if err.Errno() == exports.AILAB_NOT_FOUND {// context not exists
					return nil
				}else{
					return err
				}
			}
			jobs := mlrun.PrepareJobStatusChange()
			jobs.StatusTo  = exports.RUN_STATUS_FAILED
			_,err = tryKillRun(tx,jobs,mlrun)
			if err == nil{
				mlrun.Save(tx)
			}
			return err
		})
	}
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
		  err = checkDBQueryError(tx.Table("runs").Set("gorm:query_option","FOR UPDATE").
			  Select("id,starts,location,statistics, run_id,status,job_type,output,started,flags").
			  Joins("left join labs on runs.lab_id=labs.id").
			  Where("run_id=?",runId).Row().Scan(mlrun))

		  if err == nil && labId != 0 && labId != mlrun.ID {
			  err = exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid lab id passed for runs")
		  }
	  }
	  return
}

func tryClearLabRunsByGroup(tx*gorm.DB, labs []uint64) APIError {
	runId := ""

	err := checkDBQueryError(tx.Model(&Run{}).Unscoped().Select("run_id").Limit(1).
		Where("lab_id in ? and status < ?",labs,exports.RUN_STATUS_FAILED).Row().Scan(&runId))
	if err == nil {
		return exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE,"still have runs active !")
	}else if err.Errno() == exports.AILAB_NOT_FOUND{

		return wrapDBUpdateError(tx.Model(&Run{}).Where("lab_id in ?",labs).UpdateColumns(
			  map[string]interface{}{//@todo: clear lab directly without preclean ???
                  "deleted_at" : UnixTime{time.Now()},
                  "status":     exports.RUN_STATUS_DISCARD,
			  }),0)
	}else{
		return err
	}
}
func tryCleanLabRunsByGroup(tx*gorm.DB,labs [] uint64) (counts uint64,err APIError){
	var mlrun *BasicMLRunContext
	for _,id := range (labs) {
		mlrun,err = getBasicMLRunInfo(tx,id,"")
		if err != nil {
			return
		}
		err = execDBQuerRows(tx.Table("runs").Unscoped().Where("lab_id=? and status >= ? and deleted_at is not null",
			id,exports.RUN_STATUS_FAILED).Select(select_run_status_change), func(rows *sql.Rows) APIError {
			stats := &JobStatusChange{}
			if err   := checkDBScanError(rows.Scan(stats));err != nil {
				return err
			}
			cnt,err := tryCleanRunWithDeleted(tx,stats,mlrun)
			counts += cnt
			return err
		})
		if err == nil {	err =  mlrun.Save(tx)}
		if err != nil {	return	}
	}
	return
}
func tryKillLabRunsByGroup(tx*gorm.DB,labs []uint64) (counts uint64,err APIError){
	  var mlrun *BasicMLRunContext
	  for _,id := range (labs) {
	  	  mlrun,err = getBasicMLRunInfo(tx,id,"")
	  	  if err != nil { return }
		  err = execDBQuerRows(tx.Table("runs").Where("lab_id=? and status < ?",
		  	  id,exports.RUN_STATUS_FAILED).Select(select_run_status_change), func(rows *sql.Rows) APIError {
		  	  	stats := &JobStatusChange{}
		  	  	if err := checkDBScanError(rows.Scan(stats));err != nil {
		  	  		return err
				}
				cnt,err := tryKillRun(tx,stats,mlrun)
				counts += cnt
				return err
		  })
		  if err == nil { err =  mlrun.Save(tx) }
		  if err != nil { return}
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


func preCheckCreateRun(tx*gorm.DB, mlrun*BasicMLRunContext,req*exports.CreateJobRequest) (old*JobStatusChange,err APIError) {

	if mlrun.IsLabRun()  {//create lab run
		if (req.JobFlags & exports.RUN_FLAGS_SINGLE_INSTANCE) != 0{

			old = &JobStatusChange{}

			err = checkDBQueryError(tx.Table("runs").Select(select_run_status_change).
				Where("lab_id=? and job_type=?",mlrun.ID,req.JobType).
				Row().Scan(old))
			if err == nil{//exists singleton instance
				return old,exports.RaiseAPIError(exports.AILAB_SINGLETON_RUN_EXISTS,"singleton run exists")
			}else if err.Errno() != exports.AILAB_NOT_FOUND {
				return nil,err
			}
		}else{//track lab run starts
			mlrun.Starts ++
		}
	}else{// create nest run
		if (req.JobFlags & exports.RUN_FLAGS_SINGLE_INSTANCE) != 0{
			old = &JobStatusChange{}
			err    := checkDBQueryError(tx.Table("runs").Select(select_run_status_change).
				Where("parent=? and job_type=?",mlrun.RunId,req.JobType).
				Row().Scan(old))
			if err == nil{//exists singleton instance
				return old,exports.RaiseAPIError(exports.AILAB_SINGLETON_RUN_EXISTS,"singleton run exists")
			}else if err.Errno() != exports.AILAB_NOT_FOUND {
				return nil,err
			}
		}else{//track run nested starts
			mlrun.Started++
		}
	}
	return
}

func completeCreateRun(tx*gorm.DB, mlrun*BasicMLRunContext,req*exports.CreateJobRequest,run*Run) (err APIError){
	 if (req.JobFlags & exports.RUN_FLAGS_SINGLE_INSTANCE) == 0 {
		 if mlrun.IsLabRun() {//create lab run
	        err = wrapDBUpdateError(tx.Table("labs").Where("id=?",mlrun.ID).Update("starts",mlrun.Starts),1)
		 }else{// create nest run
		 	err = wrapDBUpdateError(tx.Table("runs").Where("run_id=?",mlrun.RunId).Update("started",mlrun.Started),1)
		 }
	 }
	 if err == nil && run.DeletedAt == nil {// asynchronously prepare data
	 	mlrun.JobStatusChange(&JobStatusChange{
			JobType:  req.JobType,
			Status:   -1,
			StatusTo: run.Status,
		})
	 	err = mlrun.Save(tx)
	 	if err == nil { err = doLogStartingRun(tx,run.RunId,run.Status) }
	 }
	 return err
}




