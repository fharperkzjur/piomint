package services

import (
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/internal/utils"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
)

// vcs connector abstract interface

type vcsConnector interface{

     init(conf configs.VcsConfig)  APIError
     // if token is empty use admin user to create for this repo
     // otherwise use that user instead , usually need LDAP like intergeration
     createRepo(name,owner ,token string) (string,string,APIError)
     deleteRepo(name,owner ,token string) APIError
     // return http_url & ssh_url according to internal config
     resolveRepoAddr(name,owner string,extranet bool) (string,string)
     // if token is empty use admin user to set deploy key for that repo
     setDeployKey(name string, pubKey string,readonly bool) APIError

	 attachRepo(ak,sk string,http_url,ssh_url string) APIError
     // do nothing now
     detachRepo(ak,sk string,http_url,ssh_url string) APIError
}

var g_vcsConnectors map[string]vcsConnector

func InitVcsConnectors() APIError{

	 g_vcsConnectors=make(map[string]vcsConnector)

     for name,conf := range(configs.GetAppConfig().VersionControl) {
	       var  driver vcsConnector = nil
     	 switch name {
           case exports.AICODE_CONNECTOR_GITEA:  driver=createGiteaConnector()
           case exports.AICODE_CONNECTOR_GITLAB: return exports.NotImplementError("gitlab connector is not implement !")
           case exports.AICODE_CONNECTOR_GITHUB: return exports.NotImplementError("gitlab connector is not implement !")
           case exports.AICODE_CONNECTOR_SVN:    return exports.NotImplementError("gitlab connector is not implement !")
           default:
         }
         if err := driver.init(conf);err != nil {
         	return err
         }
         g_vcsConnectors[name]=driver
     }
     if counts,err := models.RollBackAllInitRepos();err == nil && counts > 0{
	     logger.Warnf("rollback %d init pending repos ok !",counts)
	     return nil
     }else{
     	 return err
     }
}
func GetVcsConnector(connector string) vcsConnector {
     return g_vcsConnectors[connector]
}

func ListAppRepos(cond*exports.SearchCond,bind string,isExtranet bool) (interface{},APIError){
	codes,err := models.ListAppRepos(bind,cond)
	if codeList,ok := codes.([]models.Code);ok {
		for i,v := range(codeList) {
			vcs := GetVcsConnector(v.Connector)
			if vcs == nil {//should never happen
				continue
			}
			owner := ""
			if v.Own == exports.AICODE_OWN_TYPE_SSO {
				owner = v.Creator
			}
			codeList[i].HttpUrl,codeList[i].SshUrl =vcs.resolveRepoAddr(v.RepoId,owner,isExtranet)
		}
	}
	return codes,err
}

func QueryAppRepoDetail(repoId string,isExtranet bool)(interface{},APIError){
	code,err := models.QueryAppRepoDetail(repoId)
	if code != nil {
		if vcs := GetVcsConnector(code.Connector);vcs != nil {
			owner := ""
			if code.Own == exports.AICODE_OWN_TYPE_SSO {
				owner = code.Creator
			}
            code.HttpUrl,code.SshUrl = vcs.resolveRepoAddr(code.RepoId,owner,isExtranet)
            if len(code.SshUrl) == 0 {
            	code.AccessKey=""
            	code.SecretKey=""
            }
		}
	}
	return code,err
}

func CreateAppRepoBind(req*exports.ReqCreateRepo)(interface{},APIError){
	vcs := GetVcsConnector(req.Connector)
	if vcs == nil {
		return nil,exports.RaiseAPIError(exports.AICODE_CONNECTOR_NOT_IMPLEMENT,"invalid vcs connector:" + req.Connector)
	}
	code := &models.Code{
		RepoId:      req.RepoId,
		Description: req.Description,
		Connector:   req.Connector,
		Creator:     req.Creator,
		Meta:        nil,
		Status:      exports.AICODE_REPO_STATUS_INIT,
		HttpUrl:     req.HttpUrl,
		SshUrl:      req.SshUrl,
	}
	if len(code.HttpUrl) == 0 {// maintain this repo lifecycle
		if req.IsSsoAuth {
			code.Own=exports.AICODE_OWN_TYPE_SSO
		}else{
			code.Own=exports.AICODE_OWN_TYPE_SYS
		}
	}
	if err := models.CreateAppRepoBind(code,req.Bind,req.IsMultiBind);err != nil {
		return code.RepoId,err
	}
	if len(req.RepoId) >0 || code.Own == 0 {// ref exists repo ok
        return code.RepoId,nil
	}
	owner := ""
	token := ""
	var err APIError
	if code.Own == exports.AICODE_OWN_TYPE_SSO {
          owner=req.Creator
          token=""  //@todo: should retrive from current request token ???
          err  =  exports.RaiseAPIError(exports.AICODE_CONNECTOR_NOT_IMPLEMENT,"repo sso management not implement yet !!!")
	}else {//generate AK & SK automatically for this repo
		var e error
		code.SecretKey,code.AccessKey,e = utils.MakeSSHKeyPair()
		if e != nil {
		  err= exports.RaiseAPIError(exports.AILAB_UNKNOWN_ERROR,"ssh key generate failed :" + e.Error())
		}
	}
	if err == nil {
		_,_, err = vcs.createRepo(code.RepoId,owner,token)
	}
	if err == nil && code.Own == exports.AICODE_OWN_TYPE_SYS {
		err = vcs.setDeployKey(code.RepoId,code.AccessKey,false)
	}
	if e:= models.CompleteInitRepo(code,err == nil);e == nil {
		return code.RepoId,err
	}else if (err == nil ){
		err = e
	}
	return "",err
}

func DeleteAppRepoBind(bind , repoId string) (interface{},APIError) {
	return models.DeleteAppRepoBind(bind,repoId)
}

func DeleteRepoProcessor(event*models.Event) APIError{
	repo ,err := models.QueryAppRepoDetail(event.Data)
	if err != nil {
		if err.Errno() == exports.AILAB_NOT_FOUND {
			return nil
		}else{
			return err
		}
	}else if repo.Status != exports.AICODE_REPO_STATUS_DELETE {
		return nil
	}
	// try delete repo
	vcs := GetVcsConnector(repo.Connector)
	if vcs == nil {
		return exports.RaiseAPIError(exports.AICODE_CONNECTOR_NOT_IMPLEMENT,
			 "cleanup repo with unknown connector: " + repo.Connector)
	}
	if repo.Own > 0 {
		err = vcs.deleteRepo(repo.RepoId,repo.Creator,"")
		if err != nil && err.Errno() == exports.AICODE_CONNECTOR_NOT_FOUND {
			err = nil
		}
	}
	if err == nil {
		err= models.DeleteRepoActually(repo.RepoId)
	}
	return err
}
