
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"strconv"
)

func getCleanupFlags(extra int,clean bool) (cleanFlags int){
	     status := extra & 0xFF
	     if clean {
	     	 cleanFlags=resource_release_rollback|resource_release_job_sched|resource_release_readonly
		 }else if exports.IsRunStatusSuccess(status){
			 cleanFlags  =  resource_release_readonly | resource_release_job_sched | resource_release_commit
		 }else{
			 cleanFlags  =  resource_release_readonly | resource_release_job_sched  | resource_release_rollback
		 }
		 return
}

func InitProcessor  (event *models.Event) APIError{
	  run ,err := models.QueryRunDetail(event.Data,false,exports.AILAB_RUN_STATUS_INIT)
	  if err == nil{
	  	 return PrepareResources(run,nil,false)
	  }else if err.Errno() == exports.AILAB_NOT_FOUND{
	  	 return nil
	  }else{
	  	 return err
	  }
}

func StartProcessor(event*models.Event) APIError{
	run ,err := models.QueryRunDetail(event.Data,false,exports.AILAB_RUN_STATUS_STARTING)
	if err != nil {
		if err.Errno() == exports.AILAB_NOT_FOUND {
			return nil
		}else{
			return err
		}
	}// submit job to k8s
	status := 0
	status,err = SubmitJob(run)

	return SyncJobStatus(run.RunId,exports.AILAB_RUN_STATUS_STARTING,status,err)
}

func KillProcessor(event*models.Event) APIError{
	run ,err := models.QueryRunDetail(event.Data,false,exports.AILAB_RUN_STATUS_KILLING)
	if err != nil {
		if err.Errno() == exports.AILAB_NOT_FOUND {
			return nil
		}else{
			return err
		}
	}// submit job to k8s
	status := 0
	status,err = KillJob(run)

	return SyncJobStatus(run.RunId,exports.AILAB_RUN_STATUS_KILLING,status,err)
}

func checkReleaseJobSched(run*models.Run,cleanFlags int,filterStatus int) APIError {
	if !configs.GetAppConfig().Debug && needReleaseJobSched(cleanFlags) && !exports.HasJobCleanupWithJobSched(run.Flags) {
		err := DeleteJob(run.RunId)
		if err == nil {
            err = models.AddRunReleaseFlags(run.RunId,exports.AILAB_RUN_FLAGS_RELEASED_JOB_SCHED,filterStatus)
		}
		return err
	}
	return nil
}
func checkReleaseResources(run*models.Run,cleanFlags int,filterStatus int) (err APIError) {
	if  exports.IsJobNeedSave(run.Flags) && needReleaseSave(cleanFlags) && !exports.HasJobCleanupWithSaving(run.Flags){
		err = BatchReleaseResource(run, cleanFlags & resource_release_save )
		if err == nil {
			err = models.AddRunReleaseFlags(run.RunId,exports.AILAB_RUN_FLAGS_RELEASED_SAVING,filterStatus)
		}
	}
	if err == nil {
		if exports.IsJobNeedRefs(run.Flags) &&  needReleaseRefs(cleanFlags) && !exports.HasJobCleanupWithRefs(run.Flags){
			err = BatchReleaseResource(run,cleanFlags & resource_release_readonly)
			if err == nil{
				err = models.AddRunReleaseFlags(run.RunId,exports.AILAB_RUN_FLAGS_RELEASED_REFS,filterStatus)
			}
		}
	}
	return
}

func doCleanupResource(event*models.Event,status int) APIError {
	run ,err := models.QueryRunDetail(event.Data,exports.IsRunStatusClean(status),status)
	if err == nil{
		extra    := 0
		event.Fetch(&extra)
		cleanFlags :=getCleanupFlags(extra,exports.IsRunStatusClean(status))

		err = checkReleaseJobSched(run,cleanFlags,status)
		if err == nil {
			err = checkReleaseResources(run,cleanFlags,status)
		}
		if err == nil {
			err = models.CleanupDone(run.RunId,extra,status)
		}else if exports.IsRunStatusSuccess(extra&0xFF) && err.Errno() == exports.AILAB_CANNOT_COMMIT{
            err = models.CleanupDone(run.RunId,(extra & 0xFFFFFF00) | exports.AILAB_RUN_STATUS_SAVE_FAIL,status)
		}
		return err
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		return nil
	}else{
		return err
	}
}

func SaveProcessor(event*models.Event) APIError{
	  return doCleanupResource(event,exports.AILAB_RUN_STATUS_COMPLETING)
}

func CleanProcessor(event*models.Event) APIError{
	  return doCleanupResource(event,exports.AILAB_RUN_STATUS_CLEAN)
}

func DiscardProcessor(event*models.Event) APIError{
	run ,err := models.QueryRunDetail(event.Data,true,exports.AILAB_RUN_STATUS_DISCARDS)
	if err == nil{
		return models.DisposeRun(run)
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		return nil
	}else{
		return err
	}
}

func ClearLabProcessor(event*models.Event) APIError{

	labId ,_ := strconv.ParseUint(event.Data,0,64)
	run,err := models.SelectAnyLabRun(labId)
	if err == nil{
		err = checkReleaseResources(run,resource_release_readonly,exports.AILAB_RUN_STATUS_LAB_DISCARD)
		if err == nil {
			err = models.DisposeRun(run)
		}
		if err == nil {
			err = exports.RaiseReqWouldBlock("would clear lab in background !")
		}
		return err
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		return models.DisposeLab(labId)
	}else{
		return err
	}
}
