/* ******************************************************************************
* 2019 - present Contributed by Apulis Technology (Shenzhen) Co. LTD
*
* This program and the accompanying materials are made available under the
* terms of the MIT License, which is available at
* https://www.opensource.org/licenses/MIT
*
* See the NOTICE file distributed with this work for additional
* information regarding copyright ownership.
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
* WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
* License for the specific language governing permissions and limitations
* under the License.
*
* SPDX-License-Identifier: MIT
******************************************************************************/
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
	  run ,err := models.QueryRunDetail(event.Data,false,exports.AILAB_RUN_STATUS_INIT,false)
	  if err == nil{
	  	 notifyRunCreated(run.GetNotifierData())
	  	 return PrepareResources(run,nil,false)
	  }else if err.Errno() == exports.AILAB_NOT_FOUND{
	  	 return nil
	  }else{
	  	 return err
	  }
}

func StartProcessor(event*models.Event) APIError{
	run ,err := models.QueryRunDetail(event.Data,false,exports.AILAB_RUN_STATUS_STARTING,false)
	if err != nil {
		if err.Errno() == exports.AILAB_NOT_FOUND {
			return nil
		}else{
			return err
		}
	}// submit job to k8s
	status := 0
	//status,err = SubmitJob(run)
	status,err = SubmitJobV2(run)

	return SyncJobStatus(run.RunId,exports.AILAB_RUN_STATUS_STARTING,status,err)
}

func KillProcessor(event*models.Event) APIError{
	run ,err := models.QueryRunDetail(event.Data,false,exports.AILAB_RUN_STATUS_KILLING,false)
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
	run ,err := models.QueryRunDetail(event.Data,exports.IsRunStatusClean(status),status,false)
	if err == nil{
		extra    := run.ExtStatus
		if extra == 0 {//@mark: compatilbe with old data !!!
			event.Fetch(&extra)
		}

		cleanFlags :=getCleanupFlags(extra,exports.IsRunStatusClean(status))

		err = checkReleaseJobSched(run,cleanFlags,status)
		if err == nil {
			err = checkReleaseResources(run,cleanFlags,status)
		}
		if err == nil {
			err = models.CleanupDone(run.RunId,extra,status)
			run.Status=extra&0xFF
		}else if exports.IsRunStatusSuccess(extra&0xFF) && err.Errno() == exports.AILAB_CANNOT_COMMIT{
            err = models.CleanupDone(run.RunId,(extra & 0xFFFFFF00) | exports.AILAB_RUN_STATUS_SAVE_FAIL,status)
            run.Status=exports.AILAB_RUN_STATUS_SAVE_FAIL
		}
		//@add: notify complete msg
		if err == nil{
			notifyRunComplete(run.GetNotifierData())
		}
		return err
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		return nil
	}else{
		return err
	}
}

func WaitChildProcessor(event*models.Event) APIError {
	  return models.CheckWaitChildRuns(event.Data)
}

func SaveProcessor(event*models.Event) APIError{
	  return doCleanupResource(event,exports.AILAB_RUN_STATUS_COMPLETING)
}

func CleanProcessor(event*models.Event) APIError{
	  return doCleanupResource(event,exports.AILAB_RUN_STATUS_CLEAN)
}

func DiscardProcessor(event*models.Event) APIError{
	run ,err := models.QueryRunDetail(event.Data,true,exports.AILAB_RUN_STATUS_DISCARDS,false)
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
