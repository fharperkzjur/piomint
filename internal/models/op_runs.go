
package models

import (
	"database/sql"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
	"time"
)

type GormDBScopeHandler = func (db*gorm.DB)*gorm.DB

func tryResumeRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError){
	if run.IsStopping() || run.IsCompleting() || run.IsWaitChild(){
		return 0,exports.RaiseAPIError(exports.AILAB_INVALID_RUN_STATUS,"runs is busy cannot resume !")
	}else if run.RunActive() {//already running or started
		return 0,nil
	}else if !run.EnableResume(){
		return 0,exports.RaiseAPIError(exports.AILAB_RUN_CANNOT_RESTART,"stopped run cannot be restart !")
	}
	if run.HasInitOK() {
		run.StatusTo =  exports.AILAB_RUN_STATUS_STARTING
	}else{
		run.StatusTo = exports.AILAB_RUN_STATUS_INIT
	}
	err := change_run_status(tx,run.RunId,&run.StatusTo,0,run.Flags, mlrun.events)

	if err == nil {
		mlrun.JobStatusChange(run)
	}
	return 1,err
}

func tryKillRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError){

	if run.IsRunOnCloud() {//@todo: how to kill remote run jobs ???
		return 0,exports.NotImplementError("kill cloud run jobs not implement  !")
	}
	if !run.RunActive() || run.IsStopping(){// no need to kill
		return 0,nil
	}else if run.IsCompleting() || run.IsWaitChild() {// wrap error
		return 0,nil
	}
	if run.IsIniting() {// kill to abort immediatley
		run.StatusTo = exports.AILAB_RUN_STATUS_ABORT
	}else{
		run.StatusTo = exports.AILAB_RUN_STATUS_KILLING
	}
	err := change_run_status(tx,run.RunId,&run.StatusTo,0,run.Flags,mlrun.events)
	if err == nil {
		mlrun.JobStatusChange(run)
	}
	return 1,err
}

func tryDeleteRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError){
	if  run.RunActive() {//cannot delete a active run
		return 0,exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE,"delete run still active !")
	}
	err := wrapDBUpdateError(tx.Delete(&Run{},"run_id=?",run.RunId),1)
	if err == nil {
		run.StatusTo = exports.AILAB_RUN_STATUS_INVALID
		mlrun.JobStatusChange(run)
	}
	return 1,err
}

func tryForceDeleteRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext)(uint64,APIError){
	counts,err := tryDeleteRun(tx,run,mlrun)
	if err == nil {
		err = change_run_status(tx,run.RunId,&run.Status,Evt_clean_discard,run.Flags, mlrun.events)
	}
	return counts,err
}

func tryCleanRunWithDeleted(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError) {
	if  run.RunActive() {//cannot delete a active run , should never happen
		return 0,exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE,"clean run is still active !")
	}
	err :=change_run_status(tx,run.RunId,&run.Status,Evt_clean_discard,run.Flags, mlrun.events)
	return 1,err
}

func tryRecursiveOpRunsSelector(jobType string,handler GormDBScopeHandler)  GormDBScopeHandler  {
	return func (db *gorm.DB) *gorm.DB {
		if len(jobType) > 0 {
			db = db.Where("job_type=?",jobType)
		}
		if handler != nil {
			db = handler(db)
		}
		return db.Model(&Run{}).Select(select_run_status_change)
	}
}

func  tryRecursiveOpRuns(tx*gorm.DB, selector GormDBScopeHandler, mlrun*BasicMLRunContext,jobType string,deepScan bool ,applySelf bool,
	executor func(*gorm.DB,*JobStatusChange,*BasicMLRunContext) (uint64,APIError)) (counts uint64,err APIError){

	//inst := tx.Model(&Run{}).Select(select_run_status_change)
	//if len(jobType) > 0 {
	//	inst = inst.Where("job_type=?",jobType)
	//}
	//inst    = inst.Session(&gorm.Session{})
	inst := tx.Scopes(tryRecursiveOpRunsSelector(jobType,selector)).Session(&gorm.Session{})
	result := []JobStatusChange{}
	if mlrun.IsLabRun() {
		err = wrapDBQueryError(inst.Find(&result,"lab_id=?",mlrun.ID))
	}else if applySelf {
		result = append(result,*mlrun.PrepareJobStatusChange())
	}else{
		err = wrapDBQueryError(inst.Find(&result,"parent=?",mlrun.RunId))
	}
	if deepScan {
		for i:=0; err == nil && i<len(result);i++ {
			err = execDBQuerRows(inst.Where("parent=?",result[i].RunId),
				func(tx*gorm.DB,rows*sql.Rows)APIError{

					job := &JobStatusChange{}
					err := checkDBScanError(tx.ScanRows(rows,job))
					result = append(result,*job)
					return err

				})
		}
	}
	cnt := uint64(0)
	for i:=0; err == nil && i<len(result);i++{
		cnt,err = executor(tx,&result[i],mlrun)
		counts += cnt
	}
	// update lab statistics
	if err == nil {
		err = mlrun.Save(tx)
	}
	return
}

func  ChangeJobStatus(runId string,from,to int,msg string) APIError{
	if from == to {//should never happen
		return nil
	}
	switch to {// cannot change to internal status through this api
	case exports.AILAB_RUN_STATUS_INIT,
	     exports.AILAB_RUN_STATUS_STARTING,
	     exports.AILAB_RUN_STATUS_KILLING,
	     exports.AILAB_RUN_STATUS_WAIT_CHILD,
	     exports.AILAB_RUN_STATUS_COMPLETING,
	     exports.AILAB_RUN_STATUS_CLEAN,
	     exports.AILAB_RUN_STATUS_DISCARDS,
	     exports.AILAB_RUN_STATUS_LAB_DISCARD:
		logger.Warnf("ChangeJobStatus runId:%s to:%d logic error!!!",runId,to)
		return exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"[error] cannot change status to pending state by external !!!")
	}

	return execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{
		mlrun ,err := getBasicMLRunInfoEx(tx,0,runId,events)
		if err == nil{
			if from != exports.AILAB_RUN_STATUS_INVALID && mlrun.Status != from || mlrun.Status == to {// status context changes
				return nil
			}
			//validate logic error
			if !mlrun.StatusIsActive() {//none-active job should not change status
				logger.Warnf("ChangeJobStatus runId:%s non-active[%d==>%d] logic error !!!",runId,from,to)
				return exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR)
			}
			if mlrun.StatusIsStopping() && exports.IsRunStatusError(to) {// stop with complete error
				to = exports.AILAB_RUN_STATUS_ABORT
			}
			err = change_run_status(tx,mlrun.RunId,&to,0,mlrun.Flags, events)
			if err == nil{
				mlrun.ChangeStatusTo(to)
				err = mlrun.Save(tx)
			}
			if err == nil{
				updates := map[string]interface{}{
					"msg":msg,
				}
				if exports.IsRunStatusStopping(from) {
					updates["flags"] = gorm.Expr("flags|?",exports.AILAB_RUN_FLAGS_RELEASED_JOB_SCHED)
				}
				err = wrapDBUpdateError(tx.Table("runs").
					 Where("run_id=?",runId).
					 UpdateColumns(updates),1)
			}
			return err
		}else if err.Errno() == exports.AILAB_NOT_FOUND{//supress not found runs
			return nil
		}else{
			return err
		}
	})
}

func change_run_status(tx*gorm.DB,runId string,status *int,cleanFlags int,runFlags uint64, track EventsTrack) APIError{

	extra := 0
	var err APIError

	if *status == exports.AILAB_RUN_STATUS_DISCARDS{
		return wrapDBUpdateError(tx.Table("runs").Where("run_id=? and deleted_at != 0",runId).
			          UpdateColumn("status",*status),1)
	}else if cleanFlags == 0 {// no clean

		if exports.IsRunStatusNeedWaitChild(*status,runFlags) {
			extra,*status = *status | (Evt_clean_only<<8),exports.AILAB_RUN_STATUS_WAIT_CHILD
		}else if  exports.IsRunStatusNeedComplete(*status)  {
			extra,*status = *status | (Evt_clean_only<<8),exports.AILAB_RUN_STATUS_COMPLETING
		}
		updates := map[string]interface{}{
			"status":*status,
		}
		switch *status {
		  case exports.AILAB_RUN_STATUS_COMPLETING:
		  	      updates["flags"] = gorm.Expr("flags&?",^uint32(exports.AILAB_RUN_FLAGS_PREPARE_OK))
		  case exports.AILAB_RUN_STATUS_INIT:
		  	      updates["flags"] = gorm.Expr("flags&?",^uint32(exports.AILAB_RUN_FLAGS_RELEASED_DONE))
		  case exports.AILAB_RUN_STATUS_STARTING:
		  	      updates["start_time"]= &UnixTime{time.Now()}
		  	      updates["end_time"]  = nil
		}
		//@modify: use sperate fields to store extra status !
		if extra > 0 {
			updates["ext_status"]=extra
		}
 		  err = wrapDBUpdateError(tx.Table("runs").Where("run_id=? and deleted_at = 0",runId).
				UpdateColumns(updates),1)
	}else{
		if exports.IsRunStatusActive(*status){
			return exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"active runs cannot clean :" + runId)
		}
		extra,*status = *status | (cleanFlags<<8),exports.AILAB_RUN_STATUS_CLEAN
		err = wrapDBUpdateError(tx.Table("runs").Where("run_id=? and deleted_at != 0",runId).
			UpdateColumns(map[string]interface{}{
				"status": *status,
				"flags" : gorm.Expr("flags&?",^uint32(exports.AILAB_RUN_FLAGS_PREPARE_OK)),
				"ext_status":extra,
			}),1)
	}
	if err == nil {
		switch(*status) {
		case exports.AILAB_RUN_STATUS_INIT, exports.AILAB_RUN_STATUS_STARTING:
			err = logStartingRun(tx, runId, *status, track)
		case exports.AILAB_RUN_STATUS_KILLING:
			err = logKillRun(tx, runId, track)
		case exports.AILAB_RUN_STATUS_WAIT_CHILD:
			err = logWaitChildRun(tx,runId,track)
		case exports.AILAB_RUN_STATUS_COMPLETING:
			err = logSaveRun(tx, runId, track)
		case exports.AILAB_RUN_STATUS_CLEAN:
			err = logCleanRun(tx, runId,track)
		}
	}
	return err
}

func  AddRunReleaseFlags(runId string,flags uint64,filterStatus int) APIError{

	return wrapDBUpdateError(db.Table("runs").Where("run_id=? and status=? ",runId,filterStatus).
		UpdateColumn("flags",gorm.Expr("flags|?",flags)),1)

}

func GetLabRunStoragePath(labId uint64,runId string)(path string,err APIError) {
	 labs := uint64(0)
	 err = checkDBQueryError(db.Table("runs").Select("output,lab_id").Where("run_id=?",runId).
	 	  Row().Scan(&path,&labs))
	 if err == nil && labId !=0 && labId != labs {
		 err = exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid lab id passed for runs")
		 path= ""
	 }
	 return
}

func GetLabRunDefaultSaveModelName(runId string) (scope string,name string,err APIError){

	err = checkDBQueryError(db.Table("runs").Select("labs.bind,labs.name").Joins("left join labs on runs.lab_id=labs.id").
		Where("run_id=?",runId).Row().Scan(&scope,&name))

	return
}

func QueryLabRealStats(labId uint64, group string)(interface{},APIError) {
	if len(group) > 0 {
		return nil,exports.NotImplementError("QueryLabRealStats search for group search not implement yet !!!")
	}
	stats := &LabRunStats{}
	err := execDBQuerRows(db.Model(&Run{}).Where("lab_id=?",labId).Select("status"),
		func(tx*gorm.DB,rows *sql.Rows) APIError {
			status := 0
			if err   := checkDBScanError(rows.Scan(&status));err == nil {
				stats.Collect(status)
				return nil
			}else{
				return err
			}
		})
	if err != nil {
		return nil,err
	}else{
		return stats,nil
	}
}

type ApscRunInfo struct {
	ID          string   `json:"id"`
	Status      int      `json:"status"`
	Type        string   `json:"type"`
	ProjectName string   `json:"projectName"`
	OwnerUser	string   `json:"ownerUser"`
	OwnerOrg    string   `json:"ownerOrg"`
	UserGroup   string   `json:"userGroup"`
	CreatedAt   UnixTime `json:"createdAt"`
	StartTime   UnixTime `json:"startTime"`
	EndTime     UnixTime `json:"endTime"`
	Resources	*JsonMetaData `json:"resources"`
}

func SysGetAllRuns(req*exports.SearchCond) (data interface{} ,err APIError){
	result := []ApscRunInfo{}
	data,err = makePagedQuery(db.Model(&Run{}).Select(`run_id as id,status,job_type as type,project_name,runs.created_at,start_time,end_time,
         runs.creator as owner_user,org_name as owner_org,namespace as user_group,quota as resources`).
	     Joins("left join labs on lab_id=labs.id"),req,&result)
	return
}
