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
package models

import (
	"database/sql"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	JOB "github.com/apulis/go-business/pkg/jobscheduler"
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
	 //@mark: username may changed, use uid instead !!!
	 UserId       uint64       `json:"userId"`
	 CreatedAt UnixTime  `json:"createdAt"`
	 DeletedAt soft_delete.DeletedAt `json:"deletedAt,omitempty" gorm:"not null"`
	 Description string       `json:"description"`
	 StartTime      *UnixTime `json:"start,omitempty"`
	 EndTime        *UnixTime `json:"end,omitempty"`
	 Deadline   int64         `json:"deadline"`    // seconds , 0 no limit
	 Status     int           `json:"status"`
	 //@add: store extra status (old status) as field
	 ExtStatus  int           `json:"extStatus"`
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
	 Token      string        `json:"token,omitempty"`   // @todo:  should not return to client user ???
	 Namespace  string        `json:"-" gorm:"-"`
	 UserGroupId uint64       `json:"-" gorm:"-"`
	 LabBind     string       `json:"-" gorm:"-"`
	 ViewStatus int           `json:"viewStatus,omitempty" gorm:"-"`
	 RegisterStatus int       `json:"registerStatus,omitempty" gorm:"-"`
	 ScratchStatus  int       `json:"scratchStatus,omitempty" gorm:"-"`
}

const (
	list_runs_fields="run_id,lab_id,runs.name,num,job_type,runs.creator,runs.user_id,runs.created_at,runs.deleted_at,start_time,end_time,runs.description,status,runs.tags,runs.flags,runs.quota,msg,parent"
	select_run_status_change = "run_id,status,flags,job_type"
	select_mlrun_context = "run_id,status,ext_status,parent,job_type,output,started,flags,lab_id as id,deleted_at,endpoints"
)

type BasicMLRunContext struct{
	  BasicLabInfo
	  RunId   string   `json:"runId"`
	  Status  int      `json:"status"`
	  JobType string   `json:"jobType"`
	  Output  string   `json:"-"`
	  //track how many times nest run started
	  Started  uint64  `json:"-"`
	  Flags    uint64  `json:"flags"`
	  DeletedAt soft_delete.DeletedAt `json:"deleted"`
	  //track endpoints if exists
	  Endpoints *JsonMetaData  `json:"-"`
	  events    EventsTrack    `json:"-"`
	  Parent  string           `json:"parent"`
	  ExtStatus int            `json:"-"`
}

func (ctx*BasicMLRunContext) IsLabRun() bool {
     return len(ctx.RunId) == 0
}
func (ctx*BasicMLRunContext) IsParentRun() bool {
	return len(ctx.RunId) > 0
}
func (ctx*BasicMLRunContext) PrepareJobStatusChange() *JobStatusChange {
	 return &JobStatusChange{
		 RunId:    ctx.RunId,
		 JobType:  ctx.JobType,
		 Flags:    ctx.Flags,
		 Status:   ctx.Status,
	 }
}
func (ctx*BasicMLRunContext) WithJobStatusChange(job*JobStatusChange) *BasicMLRunContext{
	return &BasicMLRunContext{
		BasicLabInfo: ctx.BasicLabInfo,
		RunId:        job.RunId,
		Status:       job.Status,
		JobType:      job.JobType,
		Flags:        job.Flags,
		events:       ctx.events,
	}
}
func (ctx*BasicMLRunContext) StatusIsSuccess() bool {
	 return exports.IsRunStatusSuccess(ctx.Status)
}
func (ctx*BasicMLRunContext) StatusIsRunning() bool {
	return exports.IsRunStatusRunning(ctx.Status)
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
func (ctx*BasicMLRunContext) ShouldDiscard() bool{
	return exports.IsRunStatusDiscard(ctx.Status)
}
func (ctx*BasicMLRunContext) IsNativeLocalJob() bool{
	return len(ctx.RunId) > 0 && !exports.IsJobRunWithCloud(ctx.Flags)
}
func (ctx*BasicMLRunContext) IsNeedJoinJob() bool{
	return len(ctx.RunId) > 0 && exports.IsJobNeedWaitForChilds(ctx.Flags)
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
func (r*Run) StatusIsRunning() bool {
	return exports.IsRunStatusRunning(r.Status)
}
func (r*Run) StatusIsNonActive()bool {
	return exports.IsRunStatusNonActive(r.Status)
}

func (r*Run) GetNotifierData() * RunStatusNotifier {
	 return &RunStatusNotifier{
		 RunNotifyScope:   RunNotifyScope{
			 Bind:        r.LabBind,
			 UserGroupId: r.UserGroupId,
			 RunId:       r.RunId,
			 Parent:      r.Parent,
			 JobType:     r.JobType,
		 },
		 RunNotifyPayload: RunNotifyPayload{
			 CreatedAt: r.CreatedAt,
			 StartTime: r.StartTime,
			 EndTime:   r.EndTime,
			 DeletedAt: r.DeletedAt,
			 Status:    r.Status,
			 Result:    r.Result,
			 Progress:  r.Progress,
			 Msg:       r.Msg,
			 Name:      r.Name,
			 Creator:   r.Creator,
			 UserId:    r.UserId,
		 },
	 }
}

type UserResourceQuota struct{
	JOB.ResourceQuota
	Node int          `json:"node"`
	Arch string       `json:"arch"`
}



func  newLabRun(mlrun * BasicMLRunContext,req*exports.CreateJobRequest) *Run{
	  run := &Run{
		  RunId:       req.JobType + "-" + string(uuid.NewUUID()),
		  LabId:       mlrun.ID,
		  Bind:        req.JobGroup,
		  Name:        req.Name,
		  JobType:     req.JobType,
		  Parent:      mlrun.RunId,
		  Creator:     req.Creator,
		  UserId:      req.UserId,
		  Description: req.Description,
		  Arch:        req.Arch,
		  Image:       req.Engine,
		  Output:      req.OutputPath,
		  Flags:       req.JobFlags,
		  Cmd:         &JsonMetaData{},
		  Token:       req.Token,
	  }
	  run.Cmd.Save(req.Cmd)
	  if !exports.IsJobShouldSingleton(req.JobFlags) {
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
			//@mark: convert user endpoints to k8s service
			endpoints :=UserEndpointList{}
			for _,v := range(req.Endpoints) {
				patchName := v.Name
				if patchName[0] == '$' {//@mark: replace $ to - to make k8s compatible
					patchName=patchName[1:] + "-"
					if v.Name == exports.AILAB_SYS_ENDPOINT_SSH {// force sys ssh endpoint to use nodePort service
						v.NodePort = -1
						v.AccessKey=req.Creator
					}
				}
				userEndpoint := UserEndpoint{
					Name:        v.Name,
					Port:        v.Port,
					ServiceName: patchName + "-" + run.RunId,
					NodePort:    v.NodePort,
					SecureKey:   v.SecretKey,
					AccessKey:   v.AccessKey,
				}
				endpoints=append(endpoints,userEndpoint)
			}
			run.Endpoints.Save(endpoints)
	  }

	  run.Quota=&JsonMetaData{}
	  run.Quota.Save(req.Quota)
	  if req.CompactMaster {
	  	  run.Flags |= exports.AILAB_RUN_FLAGS_COMPACT_MASTER
	  }

	  return run
}

func  CreateLabRun(labId uint64,runId string,req*exports.CreateJobRequest,enableRepace bool,syncPrepare bool) (result interface{},err APIError){

	err = execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {

		mlrun ,err := getBasicMLRunInfoEx(tx,labId,runId,events)
		if err == nil {
			result, err = preCheckCreateRun(tx,mlrun,req)
			if err != nil && err.Errno() == exports.AILAB_SINGLETON_RUN_EXISTS {
				if !enableRepace {
					return err
				}
				//discard old run if possible
				//@mark: here should delete descend job also !!!
				oldRun := mlrun.WithJobStatusChange(result.(*JobStatusChange))
				_, err = tryRecursiveOpRuns(tx,nil,oldRun,"",true,true,tryForceDeleteRun)
				if err == nil {//@mark: restore lab info
					mlrun.BasicLabInfo=oldRun.BasicLabInfo
				}
				//_,err = tryForceDeleteRun(tx,result.(*JobStatusChange),mlrun)
			}
		}
		var run * Run
		if err == nil{
			if mlrun.IsParentRun() && !mlrun.StatusIsRunning() && exports.IsJobNeedWaitForChilds(req.JobFlags){
				return exports.RaiseAPIError(exports.AILAB_INVALID_RUN_STATUS,"only running job can create wait join child runs !!!")
			}
			//@add: if parent run not resides on cloud, child run cannot resides on cloud also !
			if mlrun.IsNativeLocalJob() && exports.IsJobRunWithCloud(req.JobFlags) {
				return exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"native local job cannot create child job run on clouds !!!")
			}
			run  = newLabRun(mlrun,req)
			err  = allocateLabRunStg(run,mlrun)
			if err == nil{
				//@mark: no resource request !!!
				if !exports.IsJobNeedPrepare(run.Flags){
                    run.Flags |= exports.AILAB_RUN_FLAGS_PREPARE_OK
                    run.Status = exports.AILAB_RUN_STATUS_STARTING
                    run.StartTime = &UnixTime{time.Now()}
				}else if syncPrepare {//@mark: synchronoulsy prepare create deleted run first , then change to starting when prepare success
					run.DeletedAt = soft_delete.DeletedAt(time.Now().Unix())
				}else{
					run.Status = exports.AILAB_RUN_STATUS_INIT
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
func  QueryRunDetail(runId string,unscoped bool,status int,needChild bool) (run*Run,err APIError){
	run  = &Run{}
	inst := db
	if unscoped {
		inst = inst.Unscoped()
	}
	if status > 0 {
		inst = inst.Where("status=?",status)
	}
	err =  wrapDBQueryError(inst.First(run,"run_id=?",runId))
	//if err == nil && exports.IsRunStatusStarting(status) {
	if err == nil {
		err = checkDBQueryError(db.Table("labs").Select("namespace,user_group_id,bind").
			Where("id=?",run.LabId).Row().Scan(&run.Namespace,&run.UserGroupId,&run.LabBind))
	}
	if err == nil && needChild{
		err = execDBQuerRows(db.Table("runs").Where("parent = ? and deleted_at=0 and flags&? != 0 ",
			runId,exports.AILAB_RUN_FLAGS_SINGLE_INSTANCE).
			Select("status,job_type"), func(tx *gorm.DB, rows *sql.Rows) APIError {
			status := 0
			jobType := ""
			if err:=checkDBQueryError(rows.Scan(&status,&jobType));err != nil{
				return err
			}
			switch jobType {
			case exports.AILAB_RUN_VISUALIZE:           run.ViewStatus=status
			case exports.AILAB_RUN_MODEL_REGISTER:      run.RegisterStatus=status
			case exports.AILAB_RUN_SCRATCH:             run.ScratchStatus=status
			}
			return nil
		})
	}
	if err != nil {
		run = nil
	}
	return
}

func QueryRunUserInfo(runId string) (*exports.LabRunUserInfo,APIError){
	  uinfo := &exports.LabRunUserInfo{}
	  err := wrapDBQueryError(db.Model(&Run{}).Select("labs.user_id as lab_user_id,labs.bind as lab_bind, labs.user_group_id,app,runs.user_id,lab_id,job_type").
	  	Joins("left join labs on runs.lab_id=labs.id").First(uinfo,"run_id=?",runId))
	  if err != nil {
	  	return nil,err
	  }else{
	  	return uinfo,nil
	  }
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
			  counts,err = tryRecursiveOpRuns(tx,nil,mlrun,jobType,deepScan,false,tryKillRun)
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

			if deepScan || !mlrun.IsNeedJoinJob(){
				counts,err = tryRecursiveOpRuns(tx,nil,mlrun,"",deepScan,true,tryKillRun)
			}else {//@add: support for `WAITCHILD` jobs
				counts,err = tryRecursiveOpRuns(tx,func(db*gorm.DB)*gorm.DB{
				   return db.Where("flags&? != 0",exports.AILAB_RUN_FLAGS_WAIT_CHILD)
				},mlrun,"",true,true,tryKillRun)
			}
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
				counts, err = tryRecursiveOpRuns(tx,nil,mlrun,"",true,true,tryDeleteRun)
			}else{
				counts, err = tryRecursiveOpRuns(tx,nil,mlrun,"",true,true,tryForceDeleteRun)
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
			counts,err = tryRecursiveOpRuns(inst,nil,mlrun,"",true,true,tryCleanRunWithDeleted)
		}
		return err
	})
	return
}

func ResumeLabRun(labId uint64,runId string) (mlrun *BasicMLRunContext,err APIError){

	err = execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {
		assertCheck(len(runId)>0,"ResumeLabRun must have runId")
		mlrun ,err = getBasicMLRunInfoEx(tx,labId,runId,events)
		if err == nil{
			_, err = tryRecursiveOpRuns(tx,nil,mlrun,"",false,true,tryResumeRun)
		}
		return err
	})
	return
}

func UpdateLabRun(labId uint64,runId string,req exports.RequestObject) (APIError) {
	TranslateJsonMeta(req,"progress","result")
	return wrapDBUpdateError(db.Model(&Run{}).Select("description","progress","result").
		Where("run_id=? and lab_id=?",runId,labId).Updates(req),1)
}

func PauseLabRun(labId uint64,runId string) APIError{
	 return exports.NotImplementError("PauseLabRun: not implements")
}
func ListAllLabRuns(req*exports.SearchCond,labId uint64,isNeedNestStatus bool) (data interface{} ,err APIError){

	result := []Run{}
	if len(req.Group) == 0 {//filter by lab id
          data,err = makePagedQuery(db.Select(list_runs_fields).Where("lab_id=?",labId),req,&result)
	}else{//need filter by group
          group := req.Group
          req.Group = ""
          data,err = makePagedQuery(db.Select(list_runs_fields + ",labs.bind as bind").
          	  Joins("left join labs on lab_id=labs.id").Where("labs.bind like ?",group+"%"),
          	  req,&result)
	}
	if len(result) > 0 && isNeedNestStatus {//retrieve nest child tool runs also
		jobIds :=make([]string,0,len(result))
		jobData:=make(map[string]*Run,len(result))

		for i:=0;i<len(result);i++{
			run := &result[i]
			jobIds=append(jobIds,run.RunId)
			jobData[run.RunId]=run
		}
		err = execDBQuerRows(db.Table("runs").Where("parent in ? and deleted_at=0 and flags&? != 0 ",
			jobIds,exports.AILAB_RUN_FLAGS_SINGLE_INSTANCE).
			Select("parent,status,job_type"), func(tx *gorm.DB, rows *sql.Rows) APIError {

				parent := ""
				status := 0
				jobType := ""
				if err:=checkDBQueryError(rows.Scan(&parent,&status,&jobType));err != nil{
					return err
				}
				switch jobType {
				case exports.AILAB_RUN_VISUALIZE:           jobData[parent].ViewStatus=status
				case exports.AILAB_RUN_MODEL_REGISTER:      jobData[parent].RegisterStatus=status
				case exports.AILAB_RUN_SCRATCH:             jobData[parent].ScratchStatus=status
				}
				return nil
		})
	}
	return
}
func  RollBackAllPrepareFailed() (counts uint64,err APIError){

	err= execDBTransaction(func(tx *gorm.DB, track EventsTrack) APIError {
		   fail_runs := []string{}
		   err := wrapDBQueryError(db.Table("runs").Where("deleted_at != 0 and status=0").Find(&fail_runs))
		   if err != nil {
		   	  return err
		   }
		   statusTo := exports.AILAB_RUN_STATUS_FAIL
		   for _,runId := range(fail_runs) {
			   err = change_run_status(tx,runId,&statusTo,Evt_clean_discard,0,track)
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
					"start_time":&UnixTime{time.Now()},
					"end_time"  :nil,
					"flags": gorm.Expr("flags|?",exports.AILAB_RUN_FLAGS_PREPARE_OK),
					"resource":resource,}) ,1)
			if err == nil {
				mlrun.Status=exports.AILAB_RUN_STATUS_INVALID
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
				    "start_time":&UnixTime{time.Now()},
				    "end_time"  :nil,
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

func PrepareRunFailed(runId string,msg string, isRollback bool) APIError {
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
			statusTo := exports.AILAB_RUN_STATUS_FAIL
			err =change_run_status(tx,mlrun.RunId,&statusTo,Evt_clean_discard,mlrun.Flags, mlrun.events)
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
			statusTo := exports.AILAB_RUN_STATUS_FAIL
			err = change_run_status(tx,mlrun.RunId,&statusTo,0,mlrun.Flags,mlrun.events)
			if err == nil {
				err = wrapDBUpdateError(tx.Model(&Run{RunId: runId}).Update("msg",msg),1)
			}
			if err == nil {
				mlrun.ChangeStatusTo(statusTo)
				err = mlrun.Save(tx)
			}
			return err
		})
	}
}

func checkNotifyParentWait(tx*gorm.DB,parent string,track EventsTrack) APIError {
	 if len(parent) == 0 {
	 	return nil
	 }
	 child := ""
	 err := checkDBQueryError(tx.Model(Run{}).Where("parent=? and flags&? != 0 and status < ?",parent,
		 	exports.AILAB_RUN_FLAGS_WAIT_CHILD,exports.AILAB_RUN_STATUS_MIN_NON_ACTIVE).Select("run_id").Limit(1).Row().Scan(&child))
	 if err == nil || err.Errno() != exports.AILAB_NOT_FOUND{// this is the final quit `WAIT RUN`
	 	return err
	 }
	 return logWaitChildRun(tx,parent,track)
}


func  CheckWaitChildRuns(runId string) APIError {

	return execDBTransaction(func(tx *gorm.DB, track EventsTrack) APIError {
		mlrun,err := getBasicMLRunInfoEx(tx,0,runId,track)
		if err != nil{
			if err.Errno() == exports.AILAB_NOT_FOUND {// context not exists
				return nil
			}else{
				return err
			}
		}
		if mlrun.Status != exports.AILAB_RUN_STATUS_WAIT_CHILD {
			return nil
		}
		child := ""
		err = checkDBQueryError(tx.Model(Run{}).Where("parent=? and flags&? != 0 and status < ?",mlrun.RunId,
			exports.AILAB_RUN_FLAGS_WAIT_CHILD,exports.AILAB_RUN_STATUS_MIN_NON_ACTIVE).Select("run_id").Limit(1).Row().Scan(&child))
		if err == nil || err.Errno() != exports.AILAB_NOT_FOUND{
			return err
		}
		orignalStatus := mlrun.ExtStatus & 0xFF

		err = change_run_status(tx,mlrun.RunId,&orignalStatus,0,0,mlrun.events)
		if err == nil{
			mlrun.ChangeStatusTo(orignalStatus)
			err = mlrun.Save(tx)
		}
		return err
	})
}


func  CleanupDone(runId string,extra int,filterStatus int) APIError{

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
		  if mlrun.Status != filterStatus {
		  	  return nil
		  }
		  if clean == Evt_clean_discard {
		  	  status = exports.AILAB_RUN_STATUS_DISCARDS
		  	  err = change_run_status(tx,runId,&status,0,mlrun.Flags,track)
		  	  if err == nil{
		  	  	err = tryDiscardRun(tx,runId,track)
			  }
		  }else if mlrun.DeletedAt == 0{
		  	  err = wrapDBUpdateError(tx.Table("runs").Where("run_id=?",runId).
		  	  	 UpdateColumns(map[string]interface{}{
				  "status":   status,
				  "end_time": &UnixTime{time.Now()},
			    }),1)
		  	  if err == nil && mlrun.IsNeedJoinJob() {
		  	  	  err = checkNotifyParentWait(tx, mlrun.Parent,track)
		      }
		  }else{
			  err = wrapDBUpdateError(tx.Table("runs").Where("run_id=?",runId).
				  UpdateColumn("status",status),1)
		  }
		  if err == nil && mlrun.DeletedAt == 0 && mlrun.Status != status{
		  	  mlrun.ChangeStatusTo(status)
		  	  err = mlrun.Save(tx)
		  }
		  return err
	  })
}

func  CheckIsDistributeJob(runId string) (bool ,APIError){
       quota := &JsonMetaData{}
	   if err := checkDBQueryError(db.Model(Run{}).Select("quota").Where("run_id=?",runId).Row().Scan(quota));err != nil {
	   	  return false,err
	   }else{
		  q := UserResourceQuota{}
	   	  quota.Fetch(&q)
		  return q.Node > 1 ,nil
	   }
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
		  err = wrapDBQueryError(tx.Model(&Run{}).Select(select_mlrun_context).
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
		Where("lab_id in ? and status < ?",labs,exports.AILAB_RUN_STATUS_MIN_NON_ACTIVE).Row().Scan(&runId))
	if err == nil {
		return exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE,"still have runs active !")
	}else if err.Errno() == exports.AILAB_NOT_FOUND{

		return wrapDBUpdateError(tx.Model(&Run{}).Where("lab_id in ?",labs).UpdateColumns(
			  map[string]interface{}{//@todo: clear lab directly without preclean ???
                  "deleted_at" : time.Now().Unix(),
                  "status":      exports.AILAB_RUN_STATUS_LAB_DISCARD,
			  }),0)
	}else{
		return err
	}
}
func tryCleanLabRunsByGroup(tx*gorm.DB,labs [] uint64,events EventsTrack) (counts uint64,err APIError){
	var mlrun *BasicMLRunContext
	jobs := []*JobStatusChange{}
	for _,id := range (labs) {
		mlrun,err = getBasicMLRunInfoEx(tx.Unscoped(),id,"",events)
		if err != nil {
			return
		}
		err = execDBQuerRows(tx.Table("runs").Unscoped().Where("lab_id=? and status >= ? and deleted_at !=0",
			id,exports.AILAB_RUN_STATUS_MIN_NON_ACTIVE).Select(select_run_status_change),
			        func(tx*gorm.DB,rows *sql.Rows) APIError {
						stats := &JobStatusChange{}
						if err := checkDBScanError(tx.ScanRows(rows,stats));err != nil {
							return err
						}
						jobs = append(jobs,stats)
						return nil

		})
		if err == nil{
			for _,v := range(jobs){

				cnt,err := tryCleanRunWithDeleted(tx,v,mlrun)
				if err != nil {
					return 0,err
				}
				counts += cnt
			}
		}
		if err == nil {	err =  mlrun.Save(tx)}
		if err != nil {	return	}
	}
	return
}
func tryKillLabRunsByGroup(tx*gorm.DB,labs []uint64,events EventsTrack) (counts uint64,err APIError){
	  var mlrun *BasicMLRunContext
	  jobs := []*JobStatusChange{}
	  for _,id := range (labs) {
	  	  mlrun,err = getBasicMLRunInfoEx(tx,id,"",events)
	  	  if err != nil { return }
		  err = execDBQuerRows(tx.Model(&Run{}).Where("lab_id=? and status < ?",
		  	  id,exports.AILAB_RUN_STATUS_WAIT_CHILD).Select(select_run_status_change),func(tx*gorm.DB,rows *sql.Rows) APIError {
		  	  	stats := &JobStatusChange{}
		  	  	if err := checkDBScanError(tx.ScanRows(rows,stats));err != nil {
		  	  		return err
				}
				jobs = append(jobs,stats)
				return nil
		  })
		  if err == nil{
		  	  for _,v := range(jobs) {
		  	  	cnt,err := tryKillRun(tx,v,mlrun)
		  	  	if err != nil {
		  	  		return 0,err
				}
				counts += cnt
			  }
		  }
		  if err == nil { err =  mlrun.Save(tx) }
		  if err != nil { return}
	  }
	  return
}

func tryDeleteLabRuns(tx*gorm.DB,labId uint64) APIError{
	 runId := ""

	 err := checkDBQueryError(tx.Model(&Run{}).Select("run_id").Limit(1).
	 	Where("lab_id=? and status < ?",labId,exports.AILAB_RUN_STATUS_MIN_NON_ACTIVE).Row().Scan(&runId))
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
		Where("lab_id in ? and status < ?",labs,exports.AILAB_RUN_STATUS_MIN_NON_ACTIVE).Row().Scan(&runId))
	if err == nil {
		return exports.RaiseAPIError(exports.AILAB_STILL_ACTIVE,"still have runs active !")
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		return wrapDBUpdateError(tx.Delete(&Run{},"lab_id in ?",labs),0)
	}else{
		return err
	}
}

func checkReturnSingleon(err APIError) APIError {
	if err == nil{//exists singleton instance
		return exports.RaiseAPIError(exports.AILAB_SINGLETON_RUN_EXISTS,"singleton run exists")
	}else if err.Errno() == exports.AILAB_NOT_FOUND {
		return nil
	}else{
		return err
	}
}

func preCheckCreateRun(tx*gorm.DB, mlrun*BasicMLRunContext,req*exports.CreateJobRequest) (old*JobStatusChange,err APIError) {

	if mlrun.IsLabRun()  {//create lab run
		if exports.IsJobShouldSingleton(req.JobFlags) {
			old = &JobStatusChange{}
			if exports.IsJobSingleton(req.JobFlags) {
				err = wrapDBQueryError(tx.Model(&Run{}).Select(select_run_status_change).
					First(old, "lab_id=? and job_type=?", mlrun.ID, req.JobType))
			}else{
				err = wrapDBQueryError(tx.Model(&Run{}).Select(select_run_status_change).
					First(old, "lab_id=? and job_type=? and user_id=?", mlrun.ID, req.JobType,req.UserId))
			}
			err = checkReturnSingleon(err)
		}else {
			if exports.IsJobIdentifyName(req.JobFlags){
				old = &JobStatusChange{}
				err = wrapDBQueryError(tx.Model(&Run{}).Select(select_run_status_change).
					First(old, "lab_id=? and job_type=? and name=?", mlrun.ID, req.JobType,req.Name))
				err = checkReturnSingleon(err)
			}
			mlrun.Starts ++
		}
	}else{// create nest run
		if exports.IsJobShouldSingleton(req.JobFlags) {
			old = &JobStatusChange{}
			if exports.IsJobSingleton(req.JobFlags) {
				err = wrapDBQueryError(tx.Model(&Run{}).Select(select_run_status_change).
					First(old, "parent=? and job_type=?", mlrun.RunId, req.JobType))
			}else{
				err = wrapDBQueryError(tx.Model(&Run{}).Select(select_run_status_change).
					First(old, "parent=? and job_type=? and user_id=?", mlrun.RunId, req.JobType,req.UserId))
			}
			err = checkReturnSingleon(err)
		}else {
			 if exports.IsJobIdentifyName(req.JobFlags){
				old = &JobStatusChange{}
				err = wrapDBQueryError(tx.Model(&Run{}).Select(select_run_status_change).
					First(old, "parent=? and job_type=? and name=?", mlrun.RunId, req.JobType,req.Name))
				err = checkReturnSingleon(err)
			 }
			 mlrun.Started++
		}
	}
	return
}

func completeCreateRun(tx*gorm.DB, mlrun*BasicMLRunContext,req*exports.CreateJobRequest,run*Run) (err APIError){
	 if !exports.IsJobShouldSingleton(req.JobFlags){//@modify: add singleton user runs
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
			Status:   exports.AILAB_RUN_STATUS_INVALID,
			StatusTo: run.Status,
		})
	 	if err == nil {
	 		err = logStartingRun(tx,run.RunId,run.Status,mlrun.events)
	 	}
	 }
	 return err
}




