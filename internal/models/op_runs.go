
package models

import (
	"database/sql"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
)

func tryResumeRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError){
	if run.IsStopping() || run.IsSaving(){
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
	_,err := change_run_status(tx,run.RunId,run.StatusTo,0,mlrun.events,false)

	if err == nil {
		mlrun.JobStatusChange(run)
	}
	return 1,err
}

func tryKillRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError){
	if !run.RunActive() || run.IsStopping(){// no need to kill
		return 0,nil
	}else if run.IsSaving() {// wrap error
		return 0,nil
	}
	if run.IsIniting() {// kill to abort immediatley
		run.StatusTo = exports.AILAB_RUN_STATUS_ABORT
	}else{
		run.StatusTo = exports.AILAB_RUN_STATUS_KILLING
	}
	_,err := change_run_status(tx,run.RunId,run.StatusTo,0,mlrun.events,run.NeedSave())
	if err == nil {
		mlrun.JobStatusChange(run)
	}
	return 1,err
}

func tryDeleteRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError){
	if  run.RunActive() {//cannot delete a active run
		return 0,exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE)
	}
	err := wrapDBUpdateError(tx.Delete(&Run{},"run_id=?",run.RunId),1)
	if err == nil {
		run.StatusTo = -1
		mlrun.JobStatusChange(run)
	}
	return 1,err
}

func tryForceDeleteRun(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext)(uint64,APIError){
	counts,err := tryDeleteRun(tx,run,mlrun)
	if err == nil {
		_,err = change_run_status(tx,run.RunId,run.Status,Evt_clean_and_discard,mlrun.events,run.NeedSave())
	}
	return counts,err
}

func tryCleanRunWithDeleted(tx*gorm.DB,run*JobStatusChange,mlrun*BasicMLRunContext) (uint64,APIError) {
	if  run.RunActive() {//cannot delete a active run , should never happen
		return 0,exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE,"clean run is still active !")
	}
	_,err :=change_run_status(tx,run.RunId,run.Status,Evt_clean_and_discard,mlrun.events,false)
	return 1,err
}

func  tryRecursiveOpRuns(tx*gorm.DB, mlrun*BasicMLRunContext,jobType string,deepScan bool ,applySelf bool,
	executor func(*gorm.DB,*JobStatusChange,*BasicMLRunContext) (uint64,APIError)) (counts uint64,err APIError){

	inst := tx.Model(&Run{}).Select(select_run_status_change)
	if len(jobType) > 0 {
		inst = inst.Where("job_type=?",jobType)
	}
	inst    = inst.Session(&gorm.Session{})
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
	     exports.AILAB_RUN_STATUS_SAVEING,
	     exports.AILAB_RUN_STATUS_CLEAN,
	     exports.AILAB_RUN_STATUS_DISCARDS:
		logger.Warnf("ChangeJobStatus runId:%s to:%d logic error!!!",runId,to)
		return exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR)
	}

	return execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{
		mlrun ,err := getBasicMLRunInfoEx(tx,0,runId,events)
		if err == nil{
			if from != -1 && mlrun.Status != from || mlrun.Status == to {// status context changes
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
			statusTo := 0
			statusTo,err = change_run_status(tx,mlrun.RunId,to,0,events,mlrun.NeedSave())
			if err == nil{
				mlrun.ChangeStatusTo(statusTo)
				err = mlrun.Save(tx)
			}
			return err
		}else if err.Errno() == exports.AILAB_NOT_FOUND{//supress not found runs
			return nil
		}else{
			return err
		}
	})
}

func change_run_status(tx*gorm.DB,runId string,status int,cleanFlags int,track EventsTrack,needSave bool ) ( int,  APIError){

	extra := 0
	var err APIError

	if status == exports.AILAB_RUN_STATUS_DISCARDS{//should never happen
		return status,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"change_run_status:"+runId)
	}
	if cleanFlags == 0 {// no clean

		if exports.IsRunStatusNonActive(status) && needSave {
			extra,status = status | (Evt_clean_only<<8),exports.AILAB_RUN_STATUS_SAVEING
		}
		err = wrapDBUpdateError(tx.Table("runs").Where("run_id=? and deleted_at = 0",runId).
				UpdateColumn("status",status),1)
	}else{
		if exports.IsRunStatusActive(status){
			return status,exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"active runs cannot clean :" + runId)
		}
		extra,status = status | (cleanFlags<<8),exports.AILAB_RUN_STATUS_CLEAN
		err = wrapDBUpdateError(tx.Table("runs").Where("run_id=? and deleted_at !=0 ",runId).
			UpdateColumns(map[string]interface{}{
				"status": status,
				"flags" : gorm.Expr("flags&?",^exports.AILAB_RUN_FLAGS_PREPARE_OK),
			}),1)
	}
	if err == nil {
		switch(status) {
		case exports.AILAB_RUN_STATUS_INIT, exports.AILAB_RUN_STATUS_STARTING:
			err = logStartingRun(tx, runId, status, track)
		case exports.AILAB_RUN_STATUS_KILLING:
			err = logKillRun(tx, runId, track)
		case exports.AILAB_RUN_STATUS_SAVEING:
			err = logSaveRun(tx, runId, extra, track)
		case exports.AILAB_RUN_STATUS_CLEAN:
			err = logCleanRun(tx, runId, extra, track)
		}
	}
	return status,err
}
