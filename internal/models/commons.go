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
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
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
	if t.IsZero() {
		return []byte("null"),nil
	}
	microSec := t.UnixNano() / int64(time.Millisecond)
	return []byte(strconv.FormatInt(microSec, 10)), nil
}

func (t UnixTime) Value() (driver.Value, error) {
	//var zeroTime time.Time
	//if t.Time.UnixNano() == zeroTime.UnixNano() {
	//	return nil, nil
	//}
	if t.IsZero() {
		return nil,nil
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
		return []byte("null"),nil
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
	if d == nil{
		return nil
	}
	return json.Unmarshal([]byte(d.data_str),v)
}
func (d*JsonMetaData) Save(v interface{}){
	d.data_str,_ =json.Marshal(v)
}
// GormDBDataType gorm db data type
func (JsonMetaData) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	case "mysql":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}
func TranslateJsonMeta(object exports.RequestObject, args...string)int{
	cnt := 0
	for _,key := range(args){
		value,ok   := object[key]
		if ok {
			switch ty := value.(type){
			case []interface{}:
				json_str,_ :=json.Marshal(ty)
				object[key] = JsonMetaData{json_str}
				cnt++
			case map[string]interface{}:
				json_str,_ :=json.Marshal(ty)
				object[key] = JsonMetaData{json_str}
				cnt++
			case []byte: if len(ty) >= 2 && (ty[0] == '{' || ty[0] == '['){
				object[key] = JsonMetaData{ty}
				cnt++
			}
			case string:
				if len(ty) >= 2 && (ty[0] == '{' || ty[0] == '['){
					object[key] = JsonMetaData{[]byte(ty)}
					cnt++
				}
			}
		}
	}
	return cnt
}

func checkCommonQuery(db*gorm.DB,req*exports.SearchCond)*gorm.DB{
	if len(req.Sort) > 0 {
		db = db.Order(req.Sort)
	}
	if len(req.Group) > 0 {
		if !req.MatchAll {// extaly match group
			db = db.Where("bind = ?",req.Group)
		}else{// recursively match all items
			db = db.Where("bind like ? ",req.Group + "%")
		}
	}
	if len(req.SearchWord) > 0 {
		db = db.Where("name like ? ","%"+req.SearchWord+"%")
	}
	switch req.Show{
	case exports.SHOW_WITH_DELETED:       db = db.Unscoped()
	case exports.SHOW_ONLY_DELETED:       db = db.Unscoped().Where("deleted_at != 0")
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
	if len(req.AdvanceOpFilters) > 0 {
		for cond,value := range req.AdvanceOpFilters {
			db = db.Where(cond,value)
		}
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

func execDBTransaction( executor func(*gorm.DB, EventsTrack) APIError  ) (err APIError) {

	 events := &EventCollector{}

	 err1 := db.Transaction( func(tx *gorm.DB) error {
	 	   err = executor(tx,events)
	 	   return err
	 })
	 if err1 != nil && err == nil{//execute transaction error
	 	err = checkDBUpdateError(err1)
	 }
	 if err1 == nil && events != nil{//notify async queue task work immediatley
		 for _,v := range(events.Notifiers){
		 	  notifier.Notify(&v)
		 }
		 for _,v := range(events.BackendEvents){
		 	notifier.HandleEvent(v.event,v.eventID)
		 }
	 }
	 return
}

func execDBQuerRows(tx * gorm.DB,executor func(tx*gorm.DB, rows*sql.Rows) APIError ) APIError{
	   rows , err := tx.Rows()
	   if rows == nil{
	   	  return exports.RaiseAPIError(exports.AILAB_DB_QUERY_FAILED,err.Error())
	   }
	   defer rows.Close()

	   for err == nil && rows.Next() {

	   	 err = executor(tx,rows)

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

