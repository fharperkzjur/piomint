
package models

import (
	"gorm.io/gorm"
	"encoding/json"
)

type Event struct{
	ID        uint64     `json:"id" gorm:"primary_key;auto_increment"`
	CreatedAt UnixTime   `json:"createdAt"`
	Type      string     // event type , should be enumarates
	Data      string     // text fields
	Extra     []byte     // compound informations for this event
}

const (
 	Log_evt_start_run   = "start"
	Log_evt_kill_run    = "kill"
	Log_evt_clean_run   = "clean"
)

//@mark: usually within a transaction
func  LogBackendEvent(tx * gorm.DB , ty , data string,extra interface{}) APIError{

	extraData :=[]byte{}
	if extra != nil {
		extraData,_=json.Marshal(extra)
	}
	// mark this transaction need check backend events
	tx.Set("log_events",ty)

	return wrapDBUpdateError(tx.Create(&Event{
		Type:      ty,
		Data:      data,
		Extra:     extraData,
	}),1)
}

func doLogStartRun(tx*gorm.DB, runId string) APIError {
	return LogBackendEvent(tx,Log_evt_start_run,runId,nil)
}
