package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	JOB "github.com/apulis/go-business/pkg/jobscheduler"
	"gorm.io/gorm"
	"strings"
)


//@add: wrapper for user endpoints
type UserEndpoint struct{
	Name      string       `json:"name"`
	Port     int           `json:"port"`
	ServiceName string     `json:"serviceName"`
	NodePort  int          `json:"nodePort,omitempty"`
	ServiceType string     `json:"serviceType,omitempty"`
	SecureKey string       `json:"secret_key,omitempty"`
	AccessKey string       `json:"access_key,omitempty"`
	Status    string       `json:"status,omitempty"`
}

type UserEndpointList []UserEndpoint

func (d * UserEndpoint)ToSchedulerPort() JOB.ContainerPort{
	port := JOB.ContainerPort{
		Port:        d.Port,
		TargetPort:  d.Port,
		ServiceName: d.ServiceName,
	}
	if d.NodePort != 0 {
		port.ServiceType=JOB.ServiceTypeNodePort
	}else{
		port.ServiceType=JOB.ServiceTypeClusterIP
	}
	return port
}

func (d*UserEndpoint)GetAccessEndpoint(namespace string) exports.ServiceEndpoint{

	accessPoint := exports.ServiceEndpoint{
		Name:      d.Name,
		Port:      d.Port,
		NodePort:  d.NodePort,
		Status:    d.Status,
		SecretKey: d.SecureKey,
		AccessKey: d.AccessKey,
	}
	if d.Name == exports.AILAB_SYS_ENDPOINT_SSH {//use ssh protocol
		if d.NodePort > 0 {
            accessPoint.Url=fmt.Sprintf("ssh -p %d %s@%s",d.NodePort,d.AccessKey,configs.GetAppConfig().ExtranetAddress)
		}
	}else{// use http protocol
		if d.NodePort > 0 {
			accessPoint.Url=fmt.Sprintf("%s:%d",configs.GetAppConfig().ExtranetAddress,d.NodePort)
		}else if d.NodePort == 0 {
			//@todo: hardcode service name in `default` namespace
			namespace="default"
			jsonInfo := map[string]interface{}{
				"service":d.ServiceName + "." + namespace + ".svc.cluster.local",
				"port":d.Port,
			}
			vhost ,_:= json.Marshal(jsonInfo)
            //@todo: delete `$` when generate endpoints url ......
            if d.Name == exports.AILAB_SYS_ENDPOINT_JUPYTER {
				accessPoint.Url=fmt.Sprintf("%s/endpoints/%s/%s/",configs.GetAppConfig().GatewayUrl,
					strings.ReplaceAll(d.Name,"$",""),base64.StdEncoding.EncodeToString(vhost))
			}else{
				accessPoint.Url=fmt.Sprintf("%s/endpoints/mindinsight/%s/",configs.GetAppConfig().GatewayUrl,
					base64.StdEncoding.EncodeToString(vhost))
			}
		}
	}
	return accessPoint
}

func (list *UserEndpointList)ToSchedulerPorts() []JOB.ContainerPort{
	ports := []JOB.ContainerPort{}
	for _,v := range(*list) {
		ports=append(ports,v.ToSchedulerPort())
	}
	return ports
}

func (list  UserEndpointList)FindEndpoint(name string) (*UserEndpoint,int) {
	for idx,_ := range (list) {
		if list[idx].Name == name {
			return &list[idx],idx
		}
	}
	return nil,-1
}
func (list  UserEndpointList)FindEndpointByService(serviceName string) (*UserEndpoint,int) {
	for idx,_ := range (list) {
		if list[idx].ServiceName == serviceName {
			return &list[idx],idx
		}
	}
	return nil,-1
}

func CreateUserEndpoint(labId uint64,runId string,endpoint* exports.ServiceEndpoint) (runContext *BasicMLRunContext,userEndpoint *UserEndpoint,err APIError) {

	err = execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{
		mlrun,err := getBasicMLRunInfoEx(tx,labId,runId,events)
		endpointList := UserEndpointList{}
		if err == nil {
			err1 := mlrun.Endpoints.Fetch(&endpointList)
			if err1 != nil {
				err = exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid stored endpoints")
			}
			runContext=mlrun
		}
		if err == nil {
			count := 0
			port  := endpoint.Port
			for _,v := range(endpointList) {
				if v.Name == endpoint.Name {
					userEndpoint=&v
					break
				}else if v.Name[0] != '$' {
					count++
				}
				if port != 0 && v.Port != 0 && port == v.Port {
					return exports.RaiseAPIError(exports.AILAB_ALREADY_EXISTS)
				}
			}
			if userEndpoint != nil {
				return nil
			}else if count >= exports.AILAB_USER_ENDPOINT_MAX_NUM {
				return exports.RaiseAPIError(exports.AILAB_EXCEED_LIMIT)
			}
			userEndpoint =  &UserEndpoint{
				Name:        endpoint.Name,
				Port:        endpoint.Port,
				ServiceName: endpoint.Name + "-" + runId,
				NodePort:    endpoint.NodePort,
				SecureKey:   endpoint.SecretKey,
				Status:      exports.AILAB_USER_ENDPOINT_STATUS_INIT,
			}
			endpointList = append(endpointList,*userEndpoint)
			endpointStore := &JsonMetaData{}
			endpointStore.Save(endpointList)
			err = wrapDBUpdateError(tx.Model(&Run{}).Where("run_id=?",runId).
				Update("endpoints",endpointStore),1)
		}
		return err
	})
	return
}



func DeleteUserEndpoint(labId uint64,runId string,name string) (runContext* BasicMLRunContext,userEndpoint *UserEndpoint, err APIError) {

	err = execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{
		mlrun,err := getBasicMLRunInfoEx(tx,labId,runId,events)
		endpointList := UserEndpointList{}
		if err == nil {
			err1 := mlrun.Endpoints.Fetch(&endpointList)
			if err1 != nil {
				err = exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid stored endpoints")
			}
			runContext=mlrun
		}
		if err == nil {
			userEndpoint,_ = endpointList.FindEndpoint(name)
			if userEndpoint == nil {
				return exports.NotFoundError()
			}
			if userEndpoint.Status == ""{
				return exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"static init endpoint cannot be deleted !!!")
			}
			userEndpoint.Status=exports.AILAB_USER_ENDPOINT_STATUS_STOP
			endpointStore := &JsonMetaData{}
			endpointStore.Save(endpointList)
			err = wrapDBUpdateError(tx.Model(&Run{}).Where("run_id=?",runId).
				Update("endpoints",endpointStore),1)
		}
		return err
	})
	return
}
func CompleteUserEndpoint(labId uint64,runId string, name string, oldStatus , newStatus string) APIError {
	if oldStatus == "" {
		return exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"cannot change static init endpoints !!!")
	}
	if newStatus == oldStatus {
		return nil
	}
	return  execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{
		mlrun,err := getBasicMLRunInfoEx(tx,labId,runId,events)
		endpointList := UserEndpointList{}
		if err == nil {
			err1 := mlrun.Endpoints.Fetch(&endpointList)
			if err1 != nil {
				err = exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid stored endpoints")
			}
		}
		if err == nil {
			userEndpoint,idx := endpointList.FindEndpoint(name)
			if idx < 0 {
				return exports.NotFoundError()
			}
			if userEndpoint.Status != oldStatus {
				return exports.RaiseAPIError(exports.AILAB_INVALID_ENDPOINT_STATUS,"endpoint status may change during paraellism call !")
			}
			if newStatus == "" {//remove from list
               endpointList = append(endpointList[0:idx],endpointList[idx+1:]...)
			}else{
               userEndpoint.Status = newStatus
			}
			endpointStore := &JsonMetaData{}
			endpointStore.Save(endpointList)

			err = wrapDBUpdateError(tx.Model(&Run{}).Where("run_id=?",runId).
				Update("endpoints",endpointStore),1)
		}
		return err
	})
}
//@mark:  call from message queue notifier
func ChangeEndpointStatus(runId string, serviceName string, nodePort int) APIError {
	return  execDBTransaction(func(tx*gorm.DB,events EventsTrack)APIError{
		mlrun,err := getBasicMLRunInfoEx(tx,0,runId,events)
		endpointList := UserEndpointList{}
		if err == nil {
			err1 := mlrun.Endpoints.Fetch(&endpointList)
			if err1 != nil {
				err = exports.RaiseAPIError(exports.AILAB_LOGIC_ERROR,"invalid stored endpoints")
			}
		}
		if err == nil {
			if !mlrun.StatusIsActive() {//skip non-active run service message
				logger.Warnf("runId:%s none active receive endpoint status change request nodePort:%d svc:%s, skip it !",
					 runId,nodePort,serviceName)
				return nil
			}
			userEndpoint,idx := endpointList.FindEndpointByService(serviceName)
			if idx < 0 {//service not exists
				logger.Warnf("runId:%s receive endpoint status change request nodePort:%d svc:%s, none exists skip it !",
					runId,nodePort,serviceName)
				return nil
			}
			userEndpoint.Status=exports.AILAB_USER_ENDPOINT_STATUS_READY
			userEndpoint.NodePort=nodePort
			endpointStore := &JsonMetaData{}
			endpointStore.Save(endpointList)

			err = wrapDBUpdateError(tx.Model(&Run{}).Where("run_id=?",runId).
				Update("endpoints",endpointStore),1)
		}
		return err
	})
}

