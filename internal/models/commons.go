package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
)

type APIError = exports.APIError

// For gorm time storage
type UnixTime struct {
	time.Time
}

// For gorm jsonb storage
type JsonB map[string]interface{}

// UnixTime implement gorm interfaces
func (t UnixTime) MarshalJSON() ([]byte, error) {
	microSec := t.UnixNano() / int64(time.Millisecond)
	return []byte(strconv.FormatInt(microSec, 10)), nil
}

func (t UnixTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}

func (t *UnixTime) Scan(v interface{}) error {
	value, ok := v.(time.Time)
	if ok {
		*t = UnixTime{Time: value}
		return nil
	}
	return fmt.Errorf("cannot convert %v to timestamp", v)
}

// JsonB implement gorm interfaces
func (j JsonB) Value() (driver.Value, error) {
	valueStr, err := json.Marshal(j)
	return string(valueStr), err
}

func (j *JsonB) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if err := json.Unmarshal(value.([]byte), &j); err != nil {
		return err
	}
	return nil
}

type JsonMetaData struct{
	//data map[string]interface{}
	data_str []byte
}
func (d*JsonMetaData)Empty()bool{
	return len(d.data_str) <= 2
}
func (d*JsonMetaData)MarshalJSON()([]byte,error){
	if len(d.data_str) == 0 {
		return nil,nil
	}
	return d.data_str,nil
}
func (d*JsonMetaData)UnmarshalJSON(b[]byte)error{
	if len(b) >= 2 && (b[0] == '{' || b[0]=='['){
		d.data_str = b
	}else{
		d.data_str = nil
	}
	return nil
}

func (d JsonMetaData) Value() (driver.Value, error) {
	if len(d.data_str) == 0 {
		return nil,nil
	}else{
		return d.data_str,nil
	}
}

func (d *JsonMetaData) Scan(v interface{}) error {
	switch ty:= v.(type){
	case string: d.data_str =[]byte(ty)
	case []byte: d.data_str =ty
	default:     d.data_str =nil
	}
	return nil
}

func (d*JsonMetaData) Fetch(v interface{}) error{
	return json.Unmarshal([]byte(d.data_str),v)
}
func (d*JsonMetaData) Save(v interface{}){
	d.data_str,_ =json.Marshal(v)
}

func checkCommonQuery(db*gorm.DB,req*exports.SearchCond)*gorm.DB{
	if len(req.Sort) > 0 {
		db = db.Order(req.Sort)
	}
	if len(req.SearchWord) > 0 {
		db = db.Where("name like ? ","%"+req.SearchWord+"%")
	}
	if len(req.Group) > 0 {
		if !req.MatchAll {// extaly match group
			db = db.Where("group = ?",req.Group)
		}else{// recursively match all items
			db = db.Where("group like ? ",req.Group + "%")
		}
	}
	switch req.Show{
	case exports.SHOW_WITH_DELETED:       db = db.Unscoped()
	case exports.SHOW_ONLY_DELETED:       db = db.Unscoped().Where("deleted_at is not null")
	}
	if req.PageSize != 0 {
		db = db.Limit(int(req.PageSize))
	}else{
		req.Offset=0
	}
	if req.Offset != 0{
		db = db.Offset(int(req.Offset))
	}
	//@todo:  support other query filter fields ???
	if len(req.EqualFilters) > 0{//@todo: here need to check all exists fields ???
		db = db.Where(req.EqualFilters)
	}
	return db
}
func makePagedQuery(db*gorm.DB, req*exports.SearchCond,resultSet interface{})(interface{}, APIError){
	db_inst := checkCommonQuery(db,req).Find(resultSet)
	err := wrapDBQueryError(db_inst)
	if err != nil{
		return nil,err
	}
	req.TotalCount=int64(req.Offset + uint(db_inst.RowsAffected))

	if req.PageSize == 0 || (req.Offset == 0 || db_inst.RowsAffected > 0) && db_inst.RowsAffected < int64(req.PageSize) { // none paged query
		return resultSet,nil
	}
	return resultSet, wrapDBQueryError(db_inst.Offset(-1).Limit(-1).Count(&req.TotalCount))
}
func wrapDBUpdateError(db * gorm.DB , changes int64 ) APIError{
	if err := checkDBUpdateError(db.Error);err != nil {
		return err
	}
	if changes > 0 && db.RowsAffected != changes {
		return exports.RaiseAPIError(exports.AILAB_DB_UPDATE_UNEXPECT,"unexpected update db")
	}
	return nil
}
func wrapDBQueryError(db* gorm.DB) APIError{
	return checkDBQueryError(db.Error)
}
func checkDBQueryError(err error) APIError{
	if err == nil{
		return nil
	}
	if errors.Is(err,gorm.ErrRecordNotFound) || errors.Is(err,sql.ErrNoRows){
		return exports.NotFoundError()
	}
	return exports.RaiseServerError(exports.AILAB_DB_QUERY_FAILED,err.Error())
}
func checkDBScanError(err error)APIError{
	if err == nil {
		return nil
	}else{
		return exports.RaiseServerError(exports.AILAB_DB_READ_ROWS,err.Error())
	}
}
func checkDBUpdateError(err error)APIError{
	if err == nil{
		return nil
	}
	if strings.Contains(err.Error(),"duplicate key"){
		return exports.RaiseAPIError(exports.AILAB_DB_DUPLICATE,err.Error())
	}
	return exports.RaiseServerError(exports.AILAB_DB_EXEC_FAILED,err.Error())
}

var notifier exports.NotifyBackendEvents

func execDBTransaction( executor func(tx*gorm.DB) APIError  ) (err APIError) {

	 var events interface{}

	 err1 := db.Transaction( func(tx *gorm.DB) error {
	 	   err = executor(tx)
	 	   if err == nil {
			   events, _ = tx.Get(Log_Events_Multi)
		   }
	 	   return err
	 })
	 if err1 != nil && err == nil{//execute transaction error
	 	err = checkDBUpdateError(err1)
	 }
	 if event,ok := events.(map[string]uint64);err == nil && ok {
	 	for k,v := range(event){
	 		notifier.NotifyWithEvent(k,v)
		}
	 }
	 return
}

func execDBQuerRows(tx * gorm.DB,executor func(rows*sql.Rows) APIError ) APIError{
	   rows , err := tx.Rows()
	   defer rows.Close()

	   for err == nil && rows.Next() {

	   	 err = executor(rows)

	   }
	   if err == nil {//check rows exit error code
	   	  err = rows.Err()
	   }
	   if err == nil {
	   	  return nil
	   }
	   if err1 ,ok := err.(APIError);ok {
	   	  return err1
	   }else{
		   return exports.RaiseAPIError(exports.AILAB_DB_READ_ROWS,err.Error())
	   }
}

func assertCheck(v bool ,msg string) {
	 if !v{
	 	panic(msg)
	 }
}

func SetEventNotifier(fy exports.NotifyBackendEvents){
	 notifier = fy
}

