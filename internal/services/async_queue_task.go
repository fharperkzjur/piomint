
package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
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
	return exports.NotImplementError("PreCleanProcessor")
}

func DiscardProcessor(event*models.Event) APIError{
    return exports.NotImplementError("DiscardProcessor")
}

func ClearLabProcessor(event*models.Event) APIError{
	return exports.NotImplementError("DiscardProcessor")
}
