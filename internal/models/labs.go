
package models

import (
	"database/sql"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/soft_delete"
	"time"
)

type Lab struct{
	ID        uint64         `json:"id"  gorm:"primary_key;auto_increment"`
	CreatedAt UnixTime       `json:"createdAt"`
	UpdatedAt UnixTime       `json:"updatedAt"`
	Description string       `json:"description"`
    App       string         `json:"app,omitempty" gorm:"type:varchar(64)"`
	Bind      string         `json:"group,omitempty" gorm:"uniqueIndex:lab_name_idx;not null"`                      // user defined group
	Name      string         `json:"name"  gorm:"uniqueIndex:lab_name_idx;not null;type:varchar(255)"`    // user defined name
	DeletedAt soft_delete.DeletedAt `json:"deletedAt,omitempty" gorm:"uniqueIndex:lab_name_idx"`
    Classify  string         `json:"classify,omitempty" gorm:"type:varchar(255)"`    // user defined classify
    Type      string         `json:"type" gorm:"type:varchar(32)"`                  // system defined type preset,visual,expert,autodl,scenes
    Creator   string         `json:"creator" gorm:"type:varchar(255)"`
	Starts    uint64         `json:"starts,omitempty"`
	Statistics*JsonMetaData   `json:"stats,omitempty" `     // internal statistics data snapshot
	Tags      *JsonMetaData   `json:"tags,omitempty"  `      // user defined tags
	Meta      *JsonMetaData   `json:"meta,omitempty"  `
	Location   string         `json:"location,omitempty"`                           // storage url identify for experiments output data
	Namespace  string         `json:"namespace,omitempty" gorm:"type:varchar(255)"` // system namespace this lab belong to
	OrgId        uint64       `json:"orgId,omitempty"`
	OrgName      string       `json:"orgName,omitempty" gorm:"type:varchar(255)"`
	UserGroupId  uint64       `json:"userGroupId"`
	//@mark: username may changed , use uid instead !!!
	UserId       uint64       `json:"userId"`
	ProjectName  string       `json:"projectName" gorm:"type:varchar(512)"`
}

type BasicLabInfo struct{
	ID        uint64      `json:"id"`
	// track how many times lab run started
	Starts    uint64      `json:"-"`
	Location  string      `json:"-"`
	Namespace string      `json:"namespace"`
	Statistics*JsonMetaData `json:"-"`

	stats *   JobStats
}

type LabRunStats struct{
	RunStarting  int  `json:"start,omitempty"`
	Running      int  `json:"run,omitempty"`
	Stopping     int  `json:"kill,omitempty"`
	Fails        int  `json:"fail,omitempty"`
	Errors       int  `json:"error,omitempty"`
	Aborts       int  `json:"abort,omitempty"`
	Success      int  `json:"success,omitempty"`
	Active       int  `json:"active,omitempty"`
}

type JobStatusChange struct {
	RunId      string  `json:"runId"`
	JobType    string  `json:"jobType"`
	Flags      uint64  `json:"flags"` // some flags to determine resumable jobs
	Status     int     `json:"status"`// old status
	StatusTo   int     `json:"statusTo,omitempty"`// changed new status
}

func (job*JobStatusChange)RunActive()bool{
	return exports.IsRunStatusActive(job.Status)
}
func (job*JobStatusChange)IsStopping()bool{
	return exports.IsRunStatusStopping(job.Status)
}
func (job*JobStatusChange)IsCompleting()bool{
	return exports.IsRunStatusCompleting(job.Status)
}
func (job*JobStatusChange)IsWaitChild()bool{
	return exports.IsRunStatusWaitChild(job.Status)
}
func (job*JobStatusChange)IsIniting()bool{
	return exports.IsRunStatusIniting(job.Status)
}
func (job*JobStatusChange)EnableResume()bool{
	return exports.IsJobResumable(job.Flags)
}
func (job*JobStatusChange)HasInitOK()bool{
	return exports.IsJobPrepareSuccess(job.Flags)
}
func (job*JobStatusChange)IsRunOnCloud()bool{
	return exports.IsJobRunWithCloud(job.Flags)
}

type JobStats  map[string]*LabRunStats

func (d*LabRunStats)Sum(s * LabRunStats){
     d.RunStarting += s.RunStarting
     d.Running     += s.Running
     d.Stopping    += s.Stopping
     d.Fails       += s.Fails
     d.Errors      += s.Errors
     d.Aborts      += s.Aborts
     d.Success     += s.Success
}
func (d*LabRunStats)Collect(status int) {
	 switch status{
		 case exports.AILAB_RUN_STATUS_INIT,
			 exports.AILAB_RUN_STATUS_STARTING,
			 exports.AILAB_RUN_STATUS_QUEUE,
			 exports.AILAB_RUN_STATUS_SCHEDULE:     d.RunStarting++
		 case exports.AILAB_RUN_STATUS_RUN,
		      exports.AILAB_RUN_STATUS_WAIT_CHILD,
			 exports.AILAB_RUN_STATUS_COMPLETING:   d.Running++
		 case exports.AILAB_RUN_STATUS_KILLING,
			 exports.AILAB_RUN_STATUS_STOPPING:     d.Stopping++
		 case exports.AILAB_RUN_STATUS_FAIL,exports.AILAB_RUN_STATUS_SAVE_FAIL:	d.Fails++
		 case exports.AILAB_RUN_STATUS_ERROR:       d.Errors++
		 case exports.AILAB_RUN_STATUS_ABORT:       d.Aborts++
		 case exports.AILAB_RUN_STATUS_SUCCESS:     d.Success++
	 }
	 if exports.IsRunStatusActive(status){
	 	d.Active++
	 }

}

func (d*JobStats) Sum(s JobStats) {
     for k,v := range(s) {
     	old ,_ := (*d)[k]
     	if old == nil {
     		old = &LabRunStats{}
			(*d)[k] = old
		}
		old.Sum(v)
	 }
}

func(d*JobStats)StatusChange(jobType string,from,to int){
	 var jobs * LabRunStats
	 if stats,ok := (*d)[jobType];!ok {
	 	jobs = &LabRunStats{}
		 (*d)[jobType]=jobs
	 }else{
	 	jobs = stats
	 }
	 switch(from){
	     case exports.AILAB_RUN_STATUS_INIT,
			  exports.AILAB_RUN_STATUS_STARTING,
			  exports.AILAB_RUN_STATUS_QUEUE,
			  exports.AILAB_RUN_STATUS_SCHEDULE:  jobs.RunStarting--
	     case exports.AILAB_RUN_STATUS_RUN,
	          exports.AILAB_RUN_STATUS_WAIT_CHILD,
	          exports.AILAB_RUN_STATUS_COMPLETING:   jobs.Running--
	     case exports.AILAB_RUN_STATUS_KILLING,
	          exports.AILAB_RUN_STATUS_STOPPING:  jobs.Stopping--
	     case exports.AILAB_RUN_STATUS_FAIL,exports.AILAB_RUN_STATUS_SAVE_FAIL:	jobs.Fails --
	     case exports.AILAB_RUN_STATUS_ERROR:     jobs.Errors --
	     case exports.AILAB_RUN_STATUS_ABORT:     jobs.Aborts --
	     case exports.AILAB_RUN_STATUS_SUCCESS:   jobs.Success --
	 }
	 switch(to){
		 case exports.AILAB_RUN_STATUS_INIT,
			 exports.AILAB_RUN_STATUS_STARTING,
			 exports.AILAB_RUN_STATUS_QUEUE,
			 exports.AILAB_RUN_STATUS_SCHEDULE:   jobs.RunStarting++
		 case exports.AILAB_RUN_STATUS_RUN,
		     exports.AILAB_RUN_STATUS_WAIT_CHILD,
			 exports.AILAB_RUN_STATUS_COMPLETING:     jobs.Running++
		 case exports.AILAB_RUN_STATUS_KILLING,
			 exports.AILAB_RUN_STATUS_STOPPING:   jobs.Stopping ++
		 case exports.AILAB_RUN_STATUS_FAIL,exports.AILAB_RUN_STATUS_SAVE_FAIL:    jobs.Fails    ++
		 case exports.AILAB_RUN_STATUS_ERROR:     jobs.Errors   ++
		 case exports.AILAB_RUN_STATUS_ABORT:     jobs.Aborts   ++
		 case exports.AILAB_RUN_STATUS_SUCCESS:   jobs.Success  ++
	 }
}

func (lab*BasicLabInfo) JobStatusChange(change *JobStatusChange){
	 if lab.Statistics == nil {
	 	lab.Statistics = &JsonMetaData{}
	 }
	 if lab.stats == nil {
		 lab.stats = &JobStats{}
		 lab.Statistics.Fetch(lab.stats)
	 }
	 lab.stats.StatusChange(change.JobType,change.Status,change.StatusTo)
}
func (lab*BasicLabInfo) BatchJobStatusChanges(changes []JobStatusChange) {
	  if lab.Statistics == nil {
		  lab.Statistics = &JsonMetaData{}
	  }
	  if lab.stats == nil {
		  lab.stats = &JobStats{}
		  lab.Statistics.Fetch(lab.stats)
	  }
	  for _,item := range(changes) {
	  	 lab.stats.StatusChange(item.JobType,item.Status,item.StatusTo)
	  }
}
func (lab*BasicLabInfo) Save(tx*gorm.DB) APIError{
	  if lab.stats == nil {// no change
	  	return nil
	  }
	  lab.Statistics.Save(lab.stats)
	  if tx == nil {
	  	return nil
	  }
	  return wrapDBUpdateError(tx.Table("labs").Where("id=? and deleted_at =0",lab.ID).
	  	     UpdateColumn("statistics",lab.Statistics),1)
}

func (lab*BasicLabInfo) Sum(stats*JsonMetaData) {
	  if lab.Statistics == nil {
			lab.Statistics = &JsonMetaData{}
	   }
	   if lab.stats == nil {
			lab.stats = &JobStats{}
			lab.Statistics.Fetch(lab.stats)
	   }
	   stats2 := &JobStats{}
	   stats.Fetch(stats2)
	   lab.stats.Sum(*stats2)
}


const (
	list_experiments_fields = "id,name,description,creator,created_at,updated_at,deleted_at,type,classify,statistics,tags"
	select_lab_basic_info   = "id,starts,location,statistics,namespace"
)

func ListAllLabs(req*exports.SearchCond) (interface{} ,APIError){
	return makePagedQuery(db.Select(list_experiments_fields),req, &[]Lab{})
}
func QueryLabDetail(labId uint64) (*Lab,APIError){

	lab := &Lab{}
	err := wrapDBQueryError(db.First(lab,"id = ?",labId))
	if err != nil {
		lab = nil
	}
	return lab,err
}

func UpdateLabInfo(id uint64,labs exports.RequestObject) APIError{
	_ = TranslateJsonMeta(labs,"tags","meta")
	return wrapDBUpdateError(db.Model(&Lab{ID:id}).
		Select("name","description","tags","meta").
		Updates(labs),1)
}

func create_one_lab(tx* gorm.DB,lab*Lab) APIError{
	err := wrapDBUpdateError(tx.Create(lab), 1)
	if err == nil {
		err = allocateLabStg(lab)
	}
	if err == nil{
		err = wrapDBUpdateError(tx.Model(lab).Update("location",lab.Location),1)
	}
	return err
}

func CreateLab(lab* Lab) APIError {

	return execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{
		return create_one_lab(tx,lab)
	})
}

func BatchCreateLab(labs []Lab)APIError{

	return execDBTransaction(func(tx*gorm.DB,events EventsTrack) APIError {
		for idx, _ := range(labs) {
			if err := create_one_lab(tx,&labs[idx]);err!=nil{
				return err
			}
		}
		return nil
	})
}

func DeleteLabByGroup(group string,labId uint64)(interface{},APIError){
	counts := 0
	err := execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {

		labs ,err := getLabsByGroup(tx,group,labId)
		if err == nil {
			counts = len(labs)
			if counts == 0 {
				return nil
			}
			err = tryDeleteLabRunsByGroup(tx,labs)
		}else if(err.Errno() == exports.AILAB_NOT_FOUND){// no labs need to be delete
			return nil
		}
		if err == nil{
			err = wrapDBUpdateError(tx.Delete(&Lab{},"id in ?",labs),int64(counts))
		}
		return err
	})
	return counts,err
}


func ClearLabByGroup(group string,labId uint64) (interface{},APIError) {
	counts := 0
	err := execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {

		labs ,err := getLabsByGroup(tx.Unscoped(),group,labId)
		if err == nil {
			counts = len(labs)
			if counts == 0 {
				return nil
			}
			err = tryClearLabRunsByGroup(tx,labs)
		}else if(err.Errno() == exports.AILAB_NOT_FOUND){// no labs need to be delete
			return nil
		}
		if err == nil {//@mark: cleared lab cannot be restore !!!
			err = wrapDBUpdateError(tx.Model(&Lab{}).Where("id in ?",labs).UpdateColumns(
				map[string]interface{}{
					"deleted_at" : gorm.Expr("(case when deleted_at = 0 then ? else deleted_at end)",time.Now().Unix()),
					"type":        exports.AISutdio_labs_discard,
				}),int64(counts))
		}
		if err == nil {
			for _,id := range(labs) {
				err = logClearLab(tx,id,events)
				if err != nil {
					break
				}
			}
		}
		return err
	})
	return counts,err
}

func KillLabByGroup(group string,labId uint64)(interface{},APIError) {
	counts := uint64(0)
	err := execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {

		labs ,err := getLabsByGroup(tx,group,labId)
		if err == nil {
			if len(labs) == 0 {
				return nil
			}
			counts,err = tryKillLabRunsByGroup(tx,labs,events)
		}else if(err.Errno() == exports.AILAB_NOT_FOUND){// no labs run need to be kill
			err = nil
		}
		return err
	})
	return counts,err
}

func CleanLabRunByGroup(group string,labId uint64) (interface{},APIError){
	counts := uint64(0)
	err := execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {

		labs ,err := getLabsByGroup(tx.Unscoped(),group,labId)
		if err == nil {
			if len(labs) == 0{
				return nil
			}
			counts,err = tryCleanLabRunsByGroup(tx,labs,events)
		}else if(err.Errno() == exports.AILAB_NOT_FOUND){// no labs run need to be clean
			err = nil
		}
		return err
	})
	return counts,err
}

func DeleteLab(labId uint64) APIError{
	 return execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {

	 	   lab ,err := getBasicLabInfo(tx,labId)
	 	   if err == nil {
	 	   	  err = tryDeleteLabRuns(tx,labId)
		   }
		   if err == nil{
		   	  err = wrapDBUpdateError(tx.Delete(&Lab{},"id=?",lab.ID),1)
		   }
		   return err

	 })
}

func QueryLabStats(labId uint64, group string)(interface{},APIError){

	  if labId != 0 {
		  if lab , err :=  GetBasicLabInfo(labId);err == nil{
			  return lab.Statistics,nil
		  }else{
		  	return nil,err
		  }
	  }else{
	  	  lab := &BasicLabInfo{}
	  	  err := execDBQuerRows(db.Model(&Lab{}).Where("bind like ? ",group + "%").Select("statistics"),
			  func(tx*gorm.DB,rows *sql.Rows) APIError {

			  	 stats := &JsonMetaData{}
			  	 err   := checkDBScanError(rows.Scan(stats))
				 lab.Sum(stats)
			  	 return err
			  })
	  	   if err != nil {
	  	   	  return nil,err
		   }else{
		   	  lab.Save(nil)
		   	  return lab.Statistics,nil
		   }
	  }
}

func GetBasicLabInfo(labId uint64) (lab *BasicLabInfo,err APIError) {
	lab = &BasicLabInfo{}
	err = wrapDBQueryError(db.Select(select_lab_basic_info).Model(&Lab{}).First(lab,"id=?",labId))
	return
}
// deleted lab output storage & then delete from db
func DisposeLab(labId uint64) APIError{
	lab := &BasicLabInfo{}
	err := wrapDBQueryError(db.Unscoped().Table("labs").Select("id,location").
		First(lab,"id=? and type=?",labId,exports.AISutdio_labs_discard))
	if err == nil{
		err = deleteStg(lab.Location)
		if err == nil {
			err = wrapDBUpdateError(db.Unscoped().Table("labs").Delete(lab),1)
		}
		return err
	}else if err.Errno() == exports.AILAB_NOT_FOUND{
		return nil
	}else{
		return err
	}
}

func getBasicLabInfo(tx * gorm.DB, labId uint64) (lab *BasicLabInfo,err APIError) {
	    lab = &BasicLabInfo{}
	    delete(tx.Statement.Clauses,"WHERE")
		err = wrapDBQueryError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&Lab{}).
			Select(select_lab_basic_info).
			First(lab,"id=?",labId))
		return
}

func getLabsByGroup(tx * gorm.DB, group string,labId uint64) (labs []uint64,err APIError) {
	    labs = []uint64{}
	    inst := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&Lab{})
	    if len(group) == 0 && labId == 0 {
	    	return nil,exports.ParameterError("cannot global get all labs")
		}
	    if len(group) > 0 {
	    	inst = inst.Where("bind=?",group)
		}
		if labId > 0 {
			inst = inst.Where("id=?",labId)
		}
		err = wrapDBQueryError(inst.Pluck("id",&labs))
		return
}
