
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"strconv"
)

func InitProcessor  (event *models.Event) APIError{
	  run ,err := models.QueryRunDetail(event.Data,false,exports.RUN_STATUS_INIT)
	  if err == nil{
	  	 return PrepareResources(run,false)
	  }else if err.Errno() == exports.AILAB_NOT_FOUND{
	  	 return nil
	  }else{
	  	 return err
	  }
}

func StartProcessor(event*models.Event) APIError{
	run ,err := models.QueryRunDetail(event.Data,false,exports.RUN_STATUS_STARTING)
	if err != nil {
		if err.Errno() == exports.AILAB_NOT_FOUND {
			return nil
		}else{
			return err
		}
	}// submit job to k8s
	status := 0
	status,err = SubmitJob(run)
	if err == nil {
		err = SyncJobStatus(run.RunId,status,"")
	}
	return err
}

func KillProcessor(event*models.Event) APIError{
	run ,err := models.QueryRunDetail(event.Data,false,exports.RUN_STATUS_KILLING)
	if err != nil {
		if err.Errno() == exports.AILAB_NOT_FOUND {
			return nil
		}else{
			return err
		}
	}// submit job to k8s
	status := 0
	status,err = KillJob(run)
	if err == nil {
		err = SyncJobStatus(run.RunId,status,"")
	}
	return err
}

func PreCleanProcessor(event*models.Event) APIError{

	run ,err := models.QueryRunDetail(event.Data,true,exports.RUN_STATUS_PRE_CLEAN)
	if err == nil{
		err = BatchReleaseResource(run)
		if err == nil {
			err = models.DiscardRun(run.RunId)
		}
        return err
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		return nil
	}else{
		return err
	}
}

func DiscardProcessor(event*models.Event) APIError{
	run ,err := models.QueryRunDetail(event.Data,true,exports.RUN_STATUS_DISCARD)
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
		err = BatchReleaseResource(run)
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
