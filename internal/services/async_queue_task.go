
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"strconv"
)

func getCleanupFlags(extra int,deleted bool) (cleanFlags int){
	 status := extra&0xFF
	 extra >>= 8
	 if deleted {
		 switch extra{
		 case models.Evt_clean_only:              cleanFlags=resource_release_readonly
		 case models.Evt_clean_and_discard:       cleanFlags=resource_release_readonly
		 case models.Evt_clean_create_rollback:   cleanFlags=resource_release_rollback|resource_release_readonly
		 }
	 }else if exports.IsRunStatusSuccess(status) {
	 	 cleanFlags = resource_release_commit
	 }else if exports.IsRunStatusNonActive(status){
	 	 cleanFlags = resource_release_rollback
	 }else{
         logger.Fatalf("active run cannot clean , may logic error !")
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

func doCleanRun(event*models.Event,filterStatus int) APIError{

	run ,err := models.QueryRunDetail(event.Data,exports.IsRunStatusClean(filterStatus),filterStatus)
	if err == nil{
		extra    := 0
		event.Fetch(&extra)
		cleanFlags :=getCleanupFlags(extra,run.DeletedAt!=0)
		err = BatchReleaseResource(run,cleanFlags)
		if err == nil {
			err = models.CleanupDone(run.RunId,extra)
		}
		return err
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		return nil
	}else{
		return err
	}
}

func SaveProcessor(event*models.Event) APIError{
	return doCleanRun(event,exports.AILAB_RUN_STATUS_SAVEING)
}

func CleanProcessor(event*models.Event) APIError{
    return doCleanRun(event,exports.AILAB_RUN_STATUS_CLEAN)
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
		err = BatchReleaseResource(run,resource_release_readonly)
		if err == nil {
			err = models.DisposeRun(run)
		}
		if err == nil {
			err = exports.RaiseAPIError(exports.AILAB_WOULD_BLOCK)
		}
		return err
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		return models.DisposeLab(labId)
	}else{
		return err
	}
}
