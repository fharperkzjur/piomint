
package models

import (
	"database/sql"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
	"time"
)

type Lab struct{
	ID        uint64         `json:"id"  gorm:"primary_key;auto_increment"`
	CreatedAt UnixTime       `json:"createdAt"`
	UpdatedAt UnixTime       `json:"updatedAt"`
	DeletedAt *UnixTime      `json:"deletedAt,omitempty"`
	Description string       `json:"description" gorm:"type:text"`
    App       string         `json:"app;not null"`
	Group     string         `json:"group" gorm:"uniqueIndex:lab_name_idx;not null"`    // user defined group
	Name      string         `json:"name"  gorm:"uniqueIndex:lab_name_idx;not null"`    // user defined name
    Classify  string         `json:"classify,omitempty"`    // user defined classify
    Type      string         `json:"type"`                  // system defined type preset,visual,expert,autodl,scenes
    Creator   string         `json:"creator"`
	Starts    uint64         `json:"starts"`
	Statistics*JsonMetaData   `json:"stats"`     // internal statistics data snapshot
	Tags      *JsonMetaData   `json:"tags"`      // user defined tags
	Meta      *JsonMetaData   `json:"meta"`
	Location   string         `json:"location,omitempty"`  // storage url identify for experiments output data
	Namespace  string         `json:"namespace,omitempty"` // system namespace this lab belong to
}

type BasicLabInfo struct{
	ID        uint64
	// track how many times lab run started
	Starts    uint64
	Location  string
	Statistics*JsonMetaData

	stats *   JobStats
}

type LabRunStats struct{
	RunStarting  int  `json:"start"`
	Running      int  `json:"run"`
	Stopping     int  `json:"kill"`
	Fails        int  `json:"fail"`
	Errors       int  `json:"error"`
	Aborts       int  `json:"abort"`
	Success      int  `json:"success"`
}

type JobStatusChange struct {
	RunId      string
	JobType    string
	Flags      uint64  // some flags to determine resumable jobs
	Status     int     // old status
	StatusTo   int     // changed new status
}

func (job*JobStatusChange)RunActive()bool{
	return job.Status < exports.RUN_STATUS_FAILED
}
func (job*JobStatusChange)IsStopping()bool{
	return job.Status == exports.RUN_STATUS_KILLING || job.Status == exports.RUN_STATUS_STOPPING
}
func (job*JobStatusChange)EnableResume()bool{
	return (job.Flags & exports.RUN_FLAGS_RESUMEABLE) != 0
}
func (job*JobStatusChange)IsIniting()bool{
	return job.Status == exports.RUN_STATUS_INIT
}
func (job*JobStatusChange)HasInitOK()bool{
	return (job.Flags & exports.RUN_FLAGS_PREPARE_SUCCESS) != 0
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
	     case exports.RUN_STATUS_INIT,
			  exports.RUN_STATUS_STARTING,
			  exports.RUN_STATUS_QUEUE,
			  exports.RUN_STATUS_SCHEDULE:  jobs.RunStarting--
	     case exports.RUN_STATUS_RUN,
	          exports.RUN_STATUS_SAVING:    jobs.Running--
	     case exports.RUN_STATUS_KILLING,
	          exports.RUN_STATUS_STOPPING:  jobs.Stopping--
	     case exports.RUN_STATUS_FAILED:    jobs.Fails --
	     case exports.RUN_STATUS_ERROR:     jobs.Errors --
	     case exports.RUN_STATUS_ABORT:     jobs.Aborts --
	     case exports.RUN_STATUS_SUCCESS:   jobs.Success --
	 }
	 switch(to){
		 case exports.RUN_STATUS_INIT,
			 exports.RUN_STATUS_STARTING,
			 exports.RUN_STATUS_QUEUE,
			 exports.RUN_STATUS_SCHEDULE:   jobs.RunStarting++
		 case exports.RUN_STATUS_RUN,
			 exports.RUN_STATUS_SAVING:     jobs.Running++
		 case exports.RUN_STATUS_KILLING,
			 exports.RUN_STATUS_STOPPING:   jobs.Stopping ++
		 case exports.RUN_STATUS_FAILED:    jobs.Fails    ++
		 case exports.RUN_STATUS_ERROR:     jobs.Errors   ++
		 case exports.RUN_STATUS_ABORT:     jobs.Aborts   ++
		 case exports.RUN_STATUS_SUCCESS:   jobs.Success  ++
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
	  return wrapDBUpdateError(tx.Table("labs").
	  	          Where("id=?",lab.ID).
	  	          Update("statistics",lab.Statistics),1)
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
	list_experiments_fields = "id,name,description,creator,created_at,updated_at,deleted_at,type,classify,statistics,meta"
)

func ListAllLabs(req*exports.SearchCond) (interface{} ,APIError){
	return makePagedQuery(db.Select(list_experiments_fields),req, &[]Lab{})
}
func QueryLabDetail(labId uint64) (*Lab,APIError){
	lab := &Lab{}
	err := wrapDBQueryError(db.First(lab,"id = ?",labId))
	return lab,err
}

func UpdateLabInfo(id uint64,labs exports.RequestObject) APIError{
	return wrapDBUpdateError(db.Model(&Lab{ID:id}).
		Select("name","description","tags","meta").
		Updates(labs),
		1)
}

func create_one_lab(tx* gorm.DB,lab*Lab) APIError{
	err := wrapDBUpdateError(tx.Create(lab), 1)
	if err == nil {
		err = allocateLabStg(lab)
	}
	if err == nil{
		err = wrapDBUpdateError(tx.Update("location",lab.Location),1)
	}
	return err
}

func CreateLab(lab* Lab) APIError {

	return execDBTransaction(func(tx*gorm.DB)APIError{
		return create_one_lab(tx,lab)
	})
}

func BatchCreateLab(labs []Lab)APIError{

	return execDBTransaction(func(tx*gorm.DB) APIError {
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
	err := execDBTransaction(func(tx *gorm.DB) APIError {

		labs ,err := getLabsByGroup(tx,group,labId)
		if err == nil {
			counts = len(labs)
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
	err := execDBTransaction(func(tx *gorm.DB) APIError {

		labs ,err := getLabsByGroup(tx,group,labId)
		if err == nil {
			counts = len(labs)
			err = tryClearLabRunsByGroup(tx,labs)
		}else if(err.Errno() == exports.AILAB_NOT_FOUND){// no labs need to be delete
			return nil
		}
		if err == nil {//@mark: cleared lab cannot be restore !!!
			err = wrapDBUpdateError(tx.Model(&Lab{}).Where("lab_id in ?",labs).UpdateColumns(
				map[string]interface{}{
					"deleted_at" : UnixTime{time.Now()},
					"type":        exports.AISutdio_labs_discard,
				}),int64(counts))
		}
		if err == nil {
			for _,id := range(labs) {
				err = doLogClearLab(tx,id)
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
	err := execDBTransaction(func(tx *gorm.DB) APIError {

		labs ,err := getLabsByGroup(tx,group,labId)
		if err == nil {
			counts,err = tryKillLabRunsByGroup(tx,labs)
		}else if(err.Errno() == exports.AILAB_NOT_FOUND){// no labs run need to be kill
			err = nil
		}
		return err
	})
	return counts,err
}

func CleanLabRunByGroup(group string,labId uint64) (interface{},APIError){
	counts := uint64(0)
	err := execDBTransaction(func(tx *gorm.DB) APIError {

		labs ,err := getLabsByGroup(tx,group,labId)
		if err == nil {
			counts,err = tryCleanLabRunsByGroup(tx,labs)
		}else if(err.Errno() == exports.AILAB_NOT_FOUND){// no labs run need to be clean
			err = nil
		}
		return err
	})
	return counts,err
}

func DeleteLab(labId uint64) APIError{
	 return execDBTransaction(func(tx *gorm.DB) APIError {

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
	  	  err := execDBQuerRows(db.Table("labs").Where("group like ? ",group + "%").Select("statistics"),
			  func(rows *sql.Rows) APIError {

			  	 stats := &JsonMetaData{}
			  	 err   := checkDBScanError(rows.Scan(stats))
				 lab.Sum(stats)
			  	 return err
			  })
	  	   if err != nil {
	  	   	  return nil,err
		   }else{
		   	  return lab.Statistics,nil
		   }
	  }
}

func GetBasicLabInfo(labId uint64) (lab *BasicLabInfo,err APIError) {
	lab = &BasicLabInfo{}
	err = wrapDBQueryError(db.Select("id,starts,location").First(lab,"id=?",labId))
	return
}
// deleted lab output storage & then delete from db
func DisposeLab(labId uint64) APIError{
	lab := &BasicLabInfo{}
	err := wrapDBQueryError(db.Unscoped().Select("id,starts,location").First(lab,"id=? and type=?",labId,exports.AISutdio_labs_discard))
	if err == nil{
		err = deleteStg(lab.Location)
		if err == nil {
			err = wrapDBUpdateError(db.Unscoped().Delete(lab),1)
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
		err = checkDBQueryError( tx.Set("gorm:query_option", "FOR UPDATE").Table("labs").
			Where("id=?",labId).
			Select("id,starts,location").
			Row().Scan(lab))
		return
}

func getLabsByGroup(tx * gorm.DB, group string,labId uint64) (labs []uint64,err APIError) {
	    labs = []uint64{}
	    inst := tx.Set("gorm:query_option", "FOR UPDATE").Table("labs")
	    if len(group) > 0 {
	    	inst = inst.Where("group=?",group)
		}
		if labId > 0 {
			inst = inst.Where("id=?",labId)
		}
		err = wrapDBQueryError(inst.Pluck("id",&labs))
		return
}
