package models

import (
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/apulis/go-business/pkg/wsmsg"
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
	EndTime   UnixTime              `json:"-"`
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
		jobType="未知类型"
	}
	switch  msgType {
	  case exports.AILAB_MCT_MESSAGE_TYPE_NEW:
	  	   return fmt.Sprintf("创建-%s-任务-%s",jobType,d.RunId)
	  case exports.AILAB_MCT_MESSAGE_TYPE_DEL:
	  	   return fmt.Sprintf("删除-%s-任务-%s",jobType,d.RunId)
	  case exports.AILAB_MCT_MESSAGE_TYPE_COMPLETE:
		  switch d.Status {
		  case  exports.AILAB_RUN_STATUS_SUCCESS:   return fmt.Sprintf("成功结束-%s-任务-%s",jobType,d.RunId)
		  case  exports.AILAB_RUN_STATUS_ABORT:     return fmt.Sprintf("已停止-%s-任务-%s",jobType,d.RunId)
		  case  exports.AILAB_RUN_STATUS_FAIL:      return fmt.Sprintf("失败-%s-任务-%s",jobType,d.RunId)
		  case  exports.AILAB_RUN_STATUS_SAVE_FAIL: return fmt.Sprintf("保存失败-%s-任务-%s",jobType,d.RunId)
		  default://should never happen
			  logger.Fatalf("getMsgSubject encounter unexpected run status !!!")
			  return fmt.Sprintf("未知结束-%s-任务-%s",jobType,d.RunId)
		  }
	  default:
		    return "未知消息主题"
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
		  msg.CreatedAt = d.EndTime.UTC().UnixNano()/1e6
	case exports.AILAB_MCT_MESSAGE_TYPE_DEL:
		  msg.CreatedAt = int64(d.DeletedAt*1000)
	}
	msg.Subject=d.getMsgSubject(msgType)
	return &msg
}

const (
	 select_notifier_fields = "labs.bind,user_group_id,run_id,parent,job_type,runs.created_at,runs.deleted_at,status,result,progress,msg,runs.name,runs.creator,runs.user_id"
)

func QueryRunNotifierData(runId string) (*RunStatusNotifier,APIError){
	var notifyData RunStatusNotifier
	err := wrapDBQueryError(db.Table("runs").Select(select_notifier_fields).Joins("labs on lab_id=labs.id").
		First(&notifyData,"run_id=?",runId))
	if err != nil {
		return nil,err
	}else{
		return &notifyData,nil
	}
}
