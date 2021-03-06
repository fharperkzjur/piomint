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
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/apulis/go-business/pkg/wsmsg"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type RunNotifyScope struct{
     Bind         string  `json:"bind"`
	 UserGroupId  uint64  `json:"-"`
	 RunId        string  `json:"runId"`
	 Parent       string  `json:"parent"`
	 JobType      string  `json:"jobType"`
}

type RunNotifyPayload struct{
	CreatedAt UnixTime              `json:"createdAt"`
	StartTime *UnixTime             `json:"start,omitempty"`
	EndTime   *UnixTime             `json:"end,omitempty"`
	DeletedAt soft_delete.DeletedAt `json:"deletedAt,omitempty"`
	Status    int                   `json:"status"`
	Result *  JsonMetaData          `json:"result,omitempty"`
	Progress* JsonMetaData          `json:"progress,omitempty"`
	Msg       string                `json:"msg,omitempty"`
	Name      string                `json:"name"`
	Creator   string                `json:"creator"`
	UserId    uint64                `json:"userId"`
}

type RunStatusNotifier struct{
	RunNotifyScope
	RunNotifyPayload
}

func (d*RunStatusNotifier)checkNeedStoreMsg() bool{
	switch d.JobType {
	  case exports.AILAB_RUN_TRAINING,
	       exports.AILAB_RUN_EVALUATE,
	       exports.AILAB_RUN_CODE_DEVELOP,
	       exports.AILAB_RUN_MODEL_REGISTER,
	       exports.AILAB_RUN_SCRATCH:  return true
	default:
		   return false
	}
}
func (d*RunStatusNotifier)getMsgSubject(msgType string)string{
	jobType := exports.AILAB_JOB_TYPES_ZH[d.JobType]
	if jobType == ""{
		jobType="????????????"
	}
	switch  msgType {
	  case exports.AILAB_MCT_MESSAGE_TYPE_NEW:
	  	   return fmt.Sprintf("??????-%s-??????-%s",jobType,d.RunId)
	  case exports.AILAB_MCT_MESSAGE_TYPE_DEL:
	  	   return fmt.Sprintf("??????-%s-??????-%s",jobType,d.RunId)
	  case exports.AILAB_MCT_MESSAGE_TYPE_COMPLETE:
		  switch d.Status {
		  case  exports.AILAB_RUN_STATUS_SUCCESS:   return fmt.Sprintf("????????????-%s-??????-%s",jobType,d.RunId)
		  case  exports.AILAB_RUN_STATUS_ABORT:     return fmt.Sprintf("?????????-%s-??????-%s",jobType,d.RunId)
		  case  exports.AILAB_RUN_STATUS_ERROR:     return fmt.Sprintf("??????-%s-??????-%s",jobType,d.RunId)
		  case  exports.AILAB_RUN_STATUS_FAIL:      return fmt.Sprintf("??????-%s-??????-%s",jobType,d.RunId)
		  case  exports.AILAB_RUN_STATUS_SAVE_FAIL: return fmt.Sprintf("????????????-%s-??????-%s",jobType,d.RunId)
		  default://should never happen
			  logger.Fatalf("[%s] getMsgSubject encounter unexpected run status %d !!!",d.RunId,d.Status)
			  return fmt.Sprintf("????????????-%s-??????-%s",jobType,d.RunId)
		  }
	  default:
		    return "??????????????????"
	}
}

func (d*RunStatusNotifier)GetPublishMsg(msgType string)*wsmsg.ReqPublishMessage {
	if !d.checkNeedStoreMsg() {// no need to store this message
		return nil
	}
	msg := wsmsg.ReqPublishMessage{
		Module:       "ailab",
		Classify:     wsmsg.MCT_CLASSIFY_TRAIN,
		Type:         msgType,
		Receiver:     d.UserGroupId,
		ReceiverType: wsmsg.MCT_RECV_GROUP,
		Deconstruct:  1,
	}
	switch msgType {
	case exports.AILAB_MCT_MESSAGE_TYPE_NEW:
		  msg.CreatedAt = d.CreatedAt.UTC().UnixNano()/1e6
	case exports.AILAB_MCT_MESSAGE_TYPE_COMPLETE:
		  if d.EndTime != nil {
			  msg.CreatedAt = d.EndTime.UTC().UnixNano()/1e6
		  }
	case exports.AILAB_MCT_MESSAGE_TYPE_DEL:
		  msg.CreatedAt = int64(d.DeletedAt*1000)
	}
	msg.Subject=d.getMsgSubject(msgType)
	return &msg
}

const (
	 select_notifier_fields         = "labs.bind,user_group_id,run_id,parent,job_type,runs.created_at,runs.deleted_at,status,result,progress,msg,runs.name,runs.creator,runs.user_id,start_time,end_time"
	 select_status_notifier_fields  = "labs.bind,user_group_id,parent,runs.deleted_at,status,job_type"
)

func QueryRunNotifierData(runId string) (*RunStatusNotifier,APIError){
	var notifyData RunStatusNotifier
	err := wrapDBQueryError(db.Table("runs").Select(select_notifier_fields).Joins("left join labs on lab_id=labs.id").
		First(&notifyData,"run_id=?",runId))
	if err != nil {
		return nil,err
	}else{
		return &notifyData,nil
	}
}

func QueryRunStatusNotifierData(tx*gorm.DB,runId string)(*RunStatusNotifier,APIError) {
	var notifyData RunStatusNotifier
	err := wrapDBQueryError(tx.Table("runs").Select(select_status_notifier_fields).Joins("left join labs on lab_id=labs.id").
		First(&notifyData,"run_id=?",runId))
	if err != nil {
		return nil,err
	}
	notifyData.RunId=runId
	return &notifyData,nil
}

func QueryRunDeletedNotifierData(tx*gorm.DB,jobType string,runId string)(*RunStatusNotifier,APIError){
	switch jobType {
		case exports.AILAB_RUN_TRAINING,
			exports.AILAB_RUN_EVALUATE:
		default: return nil,nil
	}
	return QueryRunStatusNotifierData(tx,runId)
}
