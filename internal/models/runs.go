package models

import (
	"database/sql"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
	"k8s.io/apimachinery/pkg/util/uuid"
	"time"
)


type Run struct{
	 RunId   string `json:"runId" gorm:"primary_key;type:varchar(255)"`
	 LabId   uint64 `json:"labId" gorm:"index;not null"`
	 Bind    string `json:"group" gorm:"index;not null"`    // user defined group , default same as lab group
	 Name    string `json:"name"  gorm:"type:varchar(255)"`
	 Started uint64 `json:"starts,omitempty"`
	 Num      uint64 `json:"num"`                                         // version number among lab or parent run
	 JobType string `json:"jobType" gorm:"type:varchar(32)"`               // system defined job types
	 Parent  string `json:"parent,omitempty" gorm:"index;type:varchar(255)"`   // created by parent run
	 Creator string `json:"creator" gorm:"type:varchar(255)"`
	 CreatedAt UnixTime  `json:"createdAt"`
	 DeletedAt soft_delete.DeletedAt `json:"deletedAt,omitempty" gorm:"not null"`
	 Description string       `json:"description"`
	 StartTime      *UnixTime `json:"start,omitempty"`
	 EndTime        *UnixTime `json:"end,omitempty"`
	 Status     int           `json:"status"`
	 Msg        string        `json:"msg"`
	 Arch       string        `json:"arch" gorm:"type:varchar(32)"`
	 Cmd        *JsonMetaData `json:"cmd,omitempty"`
	 Image      string        `json:"image,omitempty" `
	 Tags      *JsonMetaData  `json:"tags,omitempty"`
	 Config    *JsonMetaData  `json:"config,omitempty"`
	 Resource  *JsonMetaData  `json:"resource,omitempty"`
	 Envs      *JsonMetaData  `json:"envs,omitempty"`
	 Endpoints *JsonMetaData  `json:"endpoints,omitempty"`
	 Quota     *JsonMetaData  `json:"quota,omitempty"`
	 Output     string        `json:"output,omitempty"`
	 Progress   *JsonMetaData `json:"progress,omitempty"`
	 Result     *JsonMetaData `json:"result,omitempty"`
	 Flags      uint64        `json:"flags,omitempty"`   // some system defined attributes this run behaves
	 Namespace  string        `json:"-" gorm:"-"`
}

const (
	list_runs_fields="run_id,lab_id,runs.name,num,job_type,runs.creator,runs.created_at,runs.deleted_at,start_time,end_time,runs.description,status,runs.tags,msg"
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
	  DeletedAt soft_delete.DeletedAt
	  events    EventsTrack
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
func (ctx*BasicMLRunContext) StatusIsSuccess() bool {
	 return exports.IsRunStatusSuccess(ctx.Status)
}
func (ctx*BasicMLRunContext) StatusIsActive() bool {
	 return exports.IsRunStatusActive(ctx.Status)
}
func (ctx*BasicMLRunContext) StatusIsStopping() bool{
	 return exports.IsRunStatusStopping(ctx.Status)
}
func (ctx*BasicMLRunContext) StatusIsIniting() bool{
	return exports.IsRunStatusIniting(ctx.Status)
}
func (ctx*BasicMLRunContext) StatusIsClean() bool{
    return exports.IsRunStatusClean(ctx.Status)
}
func (ctx*BasicMLRunContext) StatusIsSave() bool{
	return exports.IsRunStatusSaving(ctx.Status)
}
func (ctx*BasicMLRunContext) NeedSave() bool {
	return exports.IsJobNeedSave(ctx.Flags)
}
func (ctx*BasicMLRunContext) ShouldDiscard() bool{
	return exports.IsRunStatusDiscard(ctx.Status)
}

func (ctx*BasicMLRunContext) ChangeStatusTo(status int){
	if ctx.Statistics == nil {
		ctx.Statistics = &JsonMetaData{}
	}
	if ctx.stats == nil {
		ctx.stats = &JobStats{}
		ctx.Statistics.Fetch(ctx.stats)
	}
	ctx.stats.StatusChange(ctx.JobType,ctx.Status,status)
}


func (r*Run) EnableResume() bool{
	 return exports.IsJobResumable(r.Flags)
}
func  newLabRun(mlrun * BasicMLRunContext,req*exports.CreateJobRequest) *Run{
	  run := &Run{
		  RunId:       string(uuid.NewUUID()),
		  LabId:       mlrun.ID,
		  Bind:        req.JobGroup,
		  Name:        req.Name,
		  JobType:     req.JobType,
		  Parent:      mlrun.RunId,
		  Creator:     req.Creator,
		  Description: req.Description,
		  Arch:        req.Arch,
		  Image:       req.Engine,
		  Output:      req.OutputPath,
		  Flags:       req.JobFlags,
		  Cmd:         &JsonMetaData{},
	  }
	  run.Cmd.Save(req.Cmd)
	  if !exports.IsJobSingleton(req.JobFlags) {
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

		run.Quota=&JsonMetaData{}
		run.Quota.Save(req.Quota)

	  return run
}

func  CreateLabRun(labId uint64,runId string,req*exports.CreateJobRequest,enableRepace bool,syncPrepare bool) (result interface{},err APIError){

	err = execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {

		mlrun ,err := getBasicMLRunInfoEx(tx,labId,runId,events)
		if err == nil {
			var old *JobStatusChange
			old, err = preCheckCreateRun(tx,mlrun,req)
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
                    run.Flags |= exports.AILAB_RUN_FLAGS_PREPARE_OK
                    run.Status = exports.AILAB_RUN_STATUS_STARTING
				}else if syncPrepare {//@mark: synchronoulsy prepare create deleted run first , then change to starting when prepare success
					run.DeletedAt = soft_delete.DeletedAt(time.Now().Unix())
				}
				err  = wrapDBUpdateError(tx.Create(&run),1)
			}
		}
		if err == nil{
            err = completeCreateRun(tx,mlrun,req,run)
		}
		if err == nil{
			err = mlrun.Save(tx)
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
func QueryRunStarting(runId string) (run*Run,err APIError){

	run  = &Run{RunId: runId}
	err  = wrapDBQueryError(db.Model(run).Select("runs.*,namesapce").Joins("left join labs on runs.lab_id=labs.id").Scan(run))
	return
}

func SelectAnyLabRun(labId uint64) (run*Run,err APIError){
	run = &Run{}
	return run,wrapDBQueryError(db.Unscoped().First(run,"lab_id=?",labId))
}


func  KillNestRun(labId uint64,runId string , jobType string, deepScan bool ) (counts uint64,err APIError){
	err = execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {
		  assertCheck(len(runId)>0,"KillNestRun must have runId")
		  mlrun ,err := getBasicMLRunInfoEx(tx,labId,runId,events)
		  if err == nil{
			  counts,err = tryRecursiveOpRuns(tx,mlrun,jobType,deepScan,false,tryKillRun)
		  }
		  return err
	})
	return
}
func KillLabRun(labId uint64,runId string,deepScan bool) (counts uint64,err APIError) {
	err =  execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{
		assertCheck(len(runId)>0,"KillLabRun must have runId")
		mlrun,err := getBasicMLRunInfoEx(tx,labId,runId,events)
		if err == nil {
			counts,err = tryRecursiveOpRuns(tx,mlrun,"",deepScan,true,tryKillRun)
		}
		return err
	})
	return
}
func DeleteLabRun(labId uint64,runId string,force bool) (counts uint64, err APIError){
	err = execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {

		assertCheck(len(runId)>0,"DeleteLabRun must have runId")

		mlrun ,err := getBasicMLRunInfoEx(tx,labId,runId,events)
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
	err =  execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{
		assertCheck(len(runId)>0,"CleanLabRun must have runId")
		//@mark:  here only need to traverse all deleted runs !!!
		inst := tx.Unscoped().Where("deleted_at !=0").Session(&gorm.Session{})
		mlrun,err := getBasicMLRunInfoEx(inst,labId,runId,events)
		if err == nil {
			counts,err = tryRecursiveOpRuns(inst,mlrun,"",true,true,tryCleanRunWithDeleted)
		}
		return err
	})
	return
}

func ResumeLabRun(labId uint64,runId string) (mlrun *BasicMLRunContext,err APIError){

	err = execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {
		assertCheck(len(runId)>0,"ResumeLabRun must have runId")
		mlrun ,err := getBasicMLRunInfoEx(tx,labId,runId,events)
		if err == nil{
			_, err = tryRecursiveOpRuns(tx,mlrun,"",false,true,tryResumeRun)
		}
		return err
	})
	return
}

func PauseLabRun(labId uint64,runId string) APIError{
	 return exports.NotImplementError("PauseLabRun: not implements")
}
func ListAllLabRuns(req*exports.SearchCond,labId uint64) (interface{} ,APIError){

	if len(req.Group) == 0 {//filter by lab id
          return makePagedQuery(db.Select(list_runs_fields).Where("lab_id=?",labId),req,&[]Run{})
	}else{//need filter by group
          group := req.Group
          req.Group = ""
          return makePagedQuery(db.Select(list_runs_fields + ",labs.bind as bind").
          	  Joins("left join labs on lab_id=labs.id").Where("labs.bind like ?",group+"%"),
          	  req,&[]Run{})
	}

}
func  RollBackAllPrepareFailed() (counts uint64,err APIError){

	err= execDBTransaction(func(tx *gorm.DB, track EventsTrack) APIError {
		   fail_runs := []string{}
		   err := wrapDBQueryError(db.Table("runs").Where("deleted_at != 0 and status=?",
		   	     exports.AILAB_RUN_STATUS_INIT).Find(&fail_runs))
		   if err != nil {
		   	  return err
		   }
		   for _,runId := range(fail_runs) {
			   _,err = change_run_status(tx,runId,exports.AILAB_RUN_STATUS_FAILED,Evt_clean_create_rollback,track,false)
                if err != nil {
                	return err
				}
			   counts++
		   }
		   return nil
	})
	return
}

func  PrepareRunSuccess(runId string,resource* JsonMetaData,isRollback bool) APIError{
	if isRollback {// process deleted RUN_STATUS_INIT
		return execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{

			mlrun ,err := getBasicMLRunInfoEx(tx.Unscoped().Session(&gorm.Session{}),0 ,runId,events)
			if err != nil{
				if err.Errno() == exports.AILAB_NOT_FOUND {// context not exists
					return nil
				}else{
					return err
				}
			}
			if mlrun.DeletedAt == 0 || !mlrun.StatusIsIniting() {
				return nil
			}
			err = wrapDBUpdateError( tx.Table("runs").Where("run_id=?",runId).
				UpdateColumns(map[string]interface{}{
					"status" :   exports.AILAB_RUN_STATUS_STARTING,
					"deleted_at":0,
					"flags": gorm.Expr("flags|?",exports.AILAB_RUN_FLAGS_PREPARE_OK),
					"resource":resource,}) ,1)
			if err == nil {
				mlrun.Status=-1
				mlrun.ChangeStatusTo(exports.AILAB_RUN_STATUS_STARTING)
				err = mlrun.Save(tx)
			}
			if err == nil {
				err = logStartingRun(tx,runId,exports.AILAB_RUN_STATUS_STARTING,events)
			}
			return err
		})

	}else{// process normal RUN_STATUS_INIT
		return execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{

			err := wrapDBUpdateError( tx.Table("runs").Where("run_id=? and status=? and deleted_at = 0",
				runId,exports.AILAB_RUN_STATUS_INIT).
				UpdateColumns(map[string]interface{}{
					"status" : exports.AILAB_RUN_STATUS_STARTING,
					"flags": gorm.Expr("flags|?",exports.AILAB_RUN_FLAGS_PREPARE_OK),
					"resource":resource,}) ,1)
			if err != nil && err.Errno() == exports.AILAB_DB_UPDATE_UNEXPECT {// context not exists
				return nil
			}
			if err == nil {
				err = logStartingRun(tx,runId,exports.AILAB_RUN_STATUS_STARTING,events)
			}
			return err
		})
	}
}

func PrepareRunFailed(runId string,isRollback bool) APIError {
	if isRollback {
		return execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{

			mlrun,err := getBasicMLRunInfoEx(tx.Unscoped().Session(&gorm.Session{}),0,runId,events)
			if err != nil{
				if err.Errno() == exports.AILAB_NOT_FOUND {// context not exists
					return nil
				}else{
					return err
				}
			}
			if mlrun.DeletedAt == 0 || !mlrun.StatusIsIniting() {
				return nil
			}
			_,err =change_run_status(tx,mlrun.RunId,exports.AILAB_RUN_STATUS_FAILED,Evt_clean_create_rollback,mlrun.events,false)
			return err
		})
	}else{
		return execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{

			mlrun,err := getBasicMLRunInfoEx(tx,0,runId,events)
			if err != nil{
				if err.Errno() == exports.AILAB_NOT_FOUND {// context not exists
					return nil
				}else{
					return err
				}
			}
			if !mlrun.StatusIsIniting() {
				return nil
			}
			_,err = tryKillRun(tx,mlrun.PrepareJobStatusChange(),mlrun)
			if err == nil{
				err = mlrun.Save(tx)
			}
			return err
		})
	}
}


func  CleanupDone(runId string,extra int) APIError{

	  status,clean := extra&0xFF,extra>>8

	  return execDBTransaction(func(tx *gorm.DB, track EventsTrack) APIError {
		  mlrun,err := getBasicMLRunInfoEx(tx.Unscoped().Session(&gorm.Session{}),0,runId,track)
		  if err != nil{
			  if err.Errno() == exports.AILAB_NOT_FOUND {// context not exists
				  return nil
			  }else{
				  return err
			  }
		  }
		  if clean > Evt_clean_only {
               status = exports.AILAB_RUN_STATUS_DISCARDS
		  }
		  updates := map[string]interface{}{
			  "status":status,
		  }
		  if mlrun.DeletedAt == 0 {
		  	updates["flags"] = gorm.Expr("flags&?",exports.AILAB_RUN_FLAGS_PREPARE_OK)
		  }else{
		  	updates["flags"] = gorm.Expr("(flags&?)|?",exports.AILAB_RUN_FLAGS_PREPARE_OK,exports.AILAB_RUN_FLAGS_RELEASE_DONE)
		  }
		  err = wrapDBUpdateError(tx.Table("runs").Where("run_id=?",runId).UpdateColumns(updates),1)
		  if mlrun.DeletedAt == 0 && err == nil{
		  	mlrun.ChangeStatusTo(status)
		  	err = mlrun.Save(tx)
		  }
		  if err == nil {
		  	err = logDiscardRun(tx,runId,track)
		  }
		  return err
	  })
}

func  getBasicMLRunInfoEx(tx*gorm.DB,labId uint64,runId string,events EventsTrack) (mlrun*BasicMLRunContext,err APIError){

	  mlrun = &BasicMLRunContext{events: events}
      if len(runId) == 0 {
		  lab,err := getBasicLabInfo(tx,labId)
		  if err != nil{
		  	return nil,err
		  }
		  mlrun.BasicLabInfo=*lab
	  }else{
		  err = wrapDBQueryError(tx.Model(&Run{}).Select("run_id,status,job_type,output,started,flags,lab_id as id,deleted_at").
		  	First(mlrun,"run_id=?",runId))

		  if err == nil && labId != 0 && labId != mlrun.ID {
			  err = exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid lab id passed for runs")
		  }
		  if err == nil{
			  lab,err := getBasicLabInfo(tx,mlrun.ID)
			  if err != nil{
				  return nil,err
			  }
			  mlrun.BasicLabInfo=*lab
		  }
	  }
	  return
}

func tryClearLabRunsByGroup(tx*gorm.DB, labs []uint64) APIError {
	runId := ""

	err := checkDBQueryError(tx.Model(&Run{}).Unscoped().Select("run_id").Limit(1).
		Where("lab_id in ? and status < ?",labs,exports.AILAB_RUN_STATUS_FAILED).Row().Scan(&runId))
	if err == nil {
		return exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE,"still have runs active !")
	}else if err.Errno() == exports.AILAB_NOT_FOUND{

		return wrapDBUpdateError(tx.Model(&Run{}).Where("lab_id in ?",labs).UpdateColumns(
			  map[string]interface{}{//@todo: clear lab directly without preclean ???
                  "deleted_at" : time.Now().Unix(),
                  "status":      exports.AILAB_RUN_STATUS_CLEAN,
			  }),0)
	}else{
		return err
	}
}
func tryCleanLabRunsByGroup(tx*gorm.DB,labs [] uint64,events EventsTrack) (counts uint64,err APIError){
	var mlrun *BasicMLRunContext
	for _,id := range (labs) {
		mlrun,err = getBasicMLRunInfoEx(tx,id,"",events)
		if err != nil {
			return
		}
		err = execDBQuerRows(tx.Table("runs").Unscoped().Where("lab_id=? and status >= ? and deleted_at !=0",
			id,exports.AILAB_RUN_STATUS_FAILED).Select(select_run_status_change),
			        func(tx*gorm.DB,rows *sql.Rows) APIError {
			stats := &JobStatusChange{}
			if err   := checkDBScanError(tx.ScanRows(rows,stats));err != nil {
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
func tryKillLabRunsByGroup(tx*gorm.DB,labs []uint64,events EventsTrack) (counts uint64,err APIError){
	  var mlrun *BasicMLRunContext
	  for _,id := range (labs) {
	  	  mlrun,err = getBasicMLRunInfoEx(tx,id,"",events)
	  	  if err != nil { return }
		  err = execDBQuerRows(tx.Model(&Run{}).Where("lab_id=? and status < ?",
		  	  id,exports.AILAB_RUN_STATUS_FAILED).Select(select_run_status_change),func(tx*gorm.DB,rows *sql.Rows) APIError {
		  	  	stats := &JobStatusChange{}
		  	  	if err := checkDBScanError(tx.ScanRows(rows,stats));err != nil {
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
	 	Where("lab_id=? and status < ?",labId,exports.AILAB_RUN_STATUS_FAILED).Row().Scan(&runId))
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
		Where("lab_id in ? and status < ?",labs,exports.AILAB_RUN_STATUS_FAILED).Row().Scan(&runId))
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
		if exports.IsJobSingleton(req.JobFlags) {

			old = &JobStatusChange{}
			err = wrapDBQueryError(tx.Model(&Run{}).Select(select_run_status_change).
				First(old,"lab_id=? and job_type=?",mlrun.ID,req.JobType))
			if err == nil{//exists singleton instance
				return old,exports.RaiseAPIError(exports.AILAB_SINGLETON_RUN_EXISTS,"singleton run exists")
			}else if err.Errno() == exports.AILAB_NOT_FOUND {
				return nil,nil
			}else{
				return nil,err
			}
		}else{//track lab run starts
			mlrun.Starts ++
		}
	}else{// create nest run
		if exports.IsJobSingleton(req.JobFlags){
			old = &JobStatusChange{}
			err = wrapDBQueryError(tx.Model(&Run{}).Select(select_run_status_change).
				First(old,"parent=? and job_type=?",mlrun.RunId,req.JobType))
			if err == nil{//exists singleton instance
				return old,exports.RaiseAPIError(exports.AILAB_SINGLETON_RUN_EXISTS,"singleton run exists")
			}else if err.Errno() == exports.AILAB_NOT_FOUND {
				return nil,nil
			}else{
				return nil,err
			}
		}else{//track run nested starts
			mlrun.Started++
		}
	}
	return
}

func completeCreateRun(tx*gorm.DB, mlrun*BasicMLRunContext,req*exports.CreateJobRequest,run*Run) (err APIError){
	 if !exports.IsJobSingleton(req.JobFlags) {
		 if mlrun.IsLabRun() {//create lab run
	        err = wrapDBUpdateError(tx.Table("labs").Where("id=? and deleted_at =0",mlrun.ID).
	        	Update("starts",mlrun.Starts),1)
		 }else{// create nest run
		 	err = wrapDBUpdateError(tx.Table("runs").Where("run_id=? and deleted_at = 0",mlrun.RunId).
		 		Update("started",mlrun.Started),1)
		 }
	 }
	 if err == nil && run.DeletedAt == 0  {// asynchronously prepare data
	 	mlrun.JobStatusChange(&JobStatusChange{
			JobType:  req.JobType,
			Status:   -1,
			StatusTo: run.Status,
		})
	 	if err == nil {
	 		err = logStartingRun(tx,run.RunId,run.Status,mlrun.events)
	 	}
	 }
	 return err
}




