
package models

import (
	"encoding/json"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
)

type Event struct{
	ID        uint64     `json:"id" gorm:"primary_key;auto_increment"`
	CreatedAt UnixTime   `json:"createdAt"`
	Type      string     // event type , should be enumarates
	Data      string     // text fields
	Extra     []byte     // compound informations for this event
}

const (
	Log_evt_init_run    = "init"
 	Log_evt_start_run   = "start"
	Log_evt_kill_run    = "kill"
	// check whether can clean actually
	Log_evt_pre_clean     = "clean"
	Log_evt_discard_run   = "discard"
	Log_evt_clear_lab     = "clear"
)

const (
	Log_Events_Multi = "events"
)

//@mark: usually within a transaction
func  LogBackendEvent(tx * gorm.DB , ty , data string,extra interface{}) APIError{

	extraData :=[]byte{}
	if extra != nil {
		extraData,_=json.Marshal(extra)
	}
	evt := &Event{
		Type:      ty,
		Data:      data,
		Extra:     extraData,
	}
	err := wrapDBUpdateError(tx.Create(evt),1)
	if err == nil {
		// mark this transaction need check backend events
		if v ,ok := tx.Get(Log_Events_Multi);!ok {
			tx.Set(Log_Events_Multi,map[string]uint64{
				ty:evt.ID,
			})
		}else{
			(v.(map[string]uint64))[ty]=evt.ID
		}
	}
	return err
}

func ReadBackendEvent(event *Event , ty string,  last_processed uint64) APIError{
	return wrapDBQueryError(db.First(event," id > ? and ty= ? ",last_processed,ty))
}
func RemoveBackEvent(id uint64) APIError{
	return wrapDBUpdateError(db.Delete(&Event{ID:id}),1)
}
func GetMaxBackendEventID() (eventID uint64,err APIError){
	err = checkDBScanError(db.Model(&Event{}).Select("max(id)").Row().Scan(&eventID))
	return
}

func doLogStartingRun(tx*gorm.DB, runId string,status int) APIError {
	if status == exports.RUN_STATUS_INIT {
        return LogBackendEvent(tx,Log_evt_init_run,runId,nil)
	}else{//always RUN_STATUS_START
		return LogBackendEvent(tx,Log_evt_start_run,runId,nil)
	}
}

func doLogKillRun(tx*gorm.DB,runId string) APIError{
	return LogBackendEvent(tx,Log_evt_kill_run,runId,nil)
}
//@todo:  lab runs cannot refer to another lab run ?
func doLogClearLab(tx*gorm.DB,labId uint64) APIError{
	return LogBackendEvent(tx,Log_evt_clear_lab,fmt.Sprintf("%d",labId),nil)
}

func doLogCleanRun(tx*gorm.DB,runId string) APIError {
	 return LogBackendEvent(tx,Log_evt_pre_clean,runId,nil)
}
