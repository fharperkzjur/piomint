
package services

import (
	"encoding/base64"
	"fmt"
	"github.com/apulis/bmod/ai-lab-backend/internal/configs"
	"github.com/apulis/bmod/ai-lab-backend/internal/models"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"net/http"
	"strings"
	"sync"
)

type GiteaDriver struct {
	configs.VcsConfig
	token   string       //access user token
    locker  sync.Mutex
}

const GITEA_TOKEN_NAME    = exports.AICODE_CONNECTOR_GITEA + "-token"
const GITEA_SSH_KEY_TITLE = exports.AICODE_CONNECTOR_GITEA + "-ssh"

func createGiteaConnector() vcsConnector {
      return &GiteaDriver{
	      VcsConfig: configs.VcsConfig{},
	      token:     "",
      }
}

func (d*GiteaDriver)init(conf configs.VcsConfig) APIError{

	  if len(conf.Url) > 0 {
	  	 d.Url = conf.Url
	  	 d.Host=conf.Host
	  	 d.SshHost=conf.SshHost
	  	 d.Extranet=conf.Extranet
	  }
	  if len(conf.User) > 0 { // use default user as data operator user
	  	 if len(conf.Passwd) == 0 || len(d.Url) == 0 {
	  	 	return exports.ParameterError("default user connector must have correct passwd & server host settings !!!")
	     }
	     d.User=conf.User
	     d.Passwd=conf.Passwd
	  }
	  return nil
}

func (d*GiteaDriver)createRepo(name,owner string,token string)(http_url string,ssh_url string, err APIError){
	if len(token) == 0 {//use internal server and admin user to create repo
		owner=d.User
		token,err=d.ensureUserToken()
	}
	if err != nil {
		return "","",err
	}
	url := fmt.Sprintf("%s/user/repos?token=%s",d.Url,token)
	resp := &GiteaResponse{}
	err = DoRequest(url,"POST",nil,map[string]interface{}{
		"auto_init": true,
		"default_branch": "master",
		"description": "This is the ai-code auto created repo !",
		"gitignores":  "",
		"issue_labels": "",
		"license": "",
		"name": name,
		"private": true,
		"readme": "",
		"template": false,
		"trust_model": "default",
	},resp)
	if err != nil {
		return "","",tryCheckGiteaError(err,resp.Message)
	}else{
		return resp.Http_url,resp.SSH_url,nil
	}
}

func (d*GiteaDriver)deleteRepo(name,owner string,token string) (err APIError){
	if len(token) == 0 {//use internal server and admin user to create repo
		owner=d.User
		token,err=d.ensureUserToken()
	}
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/repos/%s/%s?token=%s",d.Url,owner,name,token)
	resp := &GiteaResponse{}
	err = DoRequest(url,"DELETE",nil,nil,resp)
	if err != nil {
		return tryCheckGiteaError(err,resp.Message)
	}else{
		return nil
	}
}

func (d*GiteaDriver)setDeployKey(name string,pubKey string,readonly bool)  APIError {
	token,err := d.ensureUserToken()
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/repos/%s/%s/keys?token=%s",d.Url,d.User,name,token)
	resp := &GiteaResponse{}
	err = DoRequest(url,"POST",nil,map[string]interface{}{
		"key":   pubKey,
		"title": GITEA_SSH_KEY_TITLE,
		"read_only":readonly,
	},resp)
	if err != nil {
		return  tryCheckGiteaError(err,resp.Message)
	}else{
		return nil
	}
}

func (d*GiteaDriver)resolveRepoAddr(name,owner string,extranet bool) (string,string){
	if len(owner) == 0 {
		owner=d.User
	}
	var http_url , ssh_url string
	if !extranet {
		if len(d.Host) > 0 {
           http_url=fmt.Sprintf("http://%s/%s/%s.git",d.Host,owner,name)
		}
		if len(d.SshHost) > 0{
           if strings.ContainsRune(d.SshHost,':') {
           	 ssh_url =fmt.Sprintf("ssh://git@%s/%s/%s.git",d.SshHost,owner,name)
           }else{
           	 ssh_url =fmt.Sprintf("git@%s:%s/%s.git",d.SshHost,owner,name)
           }
		}
	}else{
		if len(d.Extranet.Host) > 0 {
			http_url=fmt.Sprintf("http://%s%s/%s/%s.git",d.Extranet.Host,d.Extranet.Prefix,owner,name)
		}
		if len(d.Extranet.SshHost) > 0{
			if strings.ContainsRune(d.Extranet.SshHost,':') {
				ssh_url =fmt.Sprintf("ssh://git@%s/%s/%s.git",d.Extranet.SshHost,owner,name)
			}else{
				ssh_url =fmt.Sprintf("git@%s:%s/%s.git",d.Extranet.SshHost,owner,name)
			}
		}
	}
	return http_url,ssh_url
}

func (d*GiteaDriver)attachRepo(ak,sk string,http_url,ssh_url string) APIError{
	return exports.NotImplementError("validate repo access not implement !!!")
}
// do nothing now
func (d*GiteaDriver)detachRepo(ak,sk string,http_url,ssh_url string) APIError{
	return nil
}

func (d*GiteaDriver)ensureUserToken() (string,APIError){
	d.locker.Lock()
	defer d.locker.Unlock()
	if len(d.token) > 0 {
		return d.token,nil
	}
	if token,err :=models.GetConfigValue(GITEA_TOKEN_NAME);err == nil{
		d.token=token // cache it
		return token,nil
	}else if err.Errno() != exports.AILAB_NOT_FOUND {
        return "",err
	}else{
		// request user token here
		token,err = createGiteaUserToken(d,GITEA_TOKEN_NAME)
		if err != nil && err.Errno() == exports.AICODE_CONNECTOR_ALREADY_EXISTS{
			err = deleteGiteaUserToken(d,GITEA_TOKEN_NAME)
			if err == nil {
				token,err = createGiteaUserToken(d,GITEA_TOKEN_NAME)
			}
		}
		if err != nil {
			return "",err
		}
		err = models.SetConfigValue(GITEA_TOKEN_NAME,token)
		//@todo: omit write error to reserve token in memory ???
		d.token=token
		return token,nil
	}
}

type GiteaResponse struct{
	Message string `json:"message"`
	Url     string `json:"url"`
	Sha1    string `json:"sha1"`
	Http_url string `json:"clone_url"`
	SSH_url  string `json:"ssh_url"`
	Html_url string `json:"html_url"`
}

func createGiteaUserToken(conf* GiteaDriver,tokenName string) (string,APIError){

	 url := fmt.Sprintf("%s/users/%s/tokens",conf.Url,conf.User)
	 resp := &GiteaResponse{}
	 err := DoRequest(url,"POST",useHttpBasicAuth(conf.User,conf.Passwd,nil),map[string]string{
            "name":tokenName,
	 },resp)
	 if err != nil {
		 return "",tryCheckGiteaError(err,resp.Message)
	 }else{
	 	return resp.Sha1,nil
	 }
}
func deleteGiteaUserToken(conf* GiteaDriver,tokenName string) APIError{

	url := fmt.Sprintf("%s/users/%s/tokens/%s",conf.Url,conf.User,tokenName)
	resp := &GiteaResponse{}
	err := DoRequest(url,"DELETE",useHttpBasicAuth(conf.User,conf.Passwd,nil),map[string]string{
		"name":tokenName,
	},resp)
	if err != nil {
		return tryCheckGiteaError(err,resp.Message)
	}else{
		return nil
	}
}
func useHttpBasicAuth(user,passwd string,headers map[string]string) map[string]string{
	if headers == nil {
		headers=make(map[string]string)
	}
	headers["Authorization"]="Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+passwd))
	return headers
}
func tryCheckGiteaError(err APIError,msg string) APIError {
	e := err.(*exports.APIException)
	if len(msg) > 0 {
		e.Msg=msg
	}
	if e.StatusCode == http.StatusUnauthorized {
		e.Code = exports.AICODE_CONNECTOR_AUTH_ERROR
	}else if e.StatusCode == http.StatusNotFound {
		e.Code = exports.AICODE_CONNECTOR_NOT_FOUND
	}else if strings.Contains(e.Msg,"already"){
		e.Code = exports.AICODE_CONNECTOR_ALREADY_EXISTS
	}else if e.Code  == exports.AILAB_REMOTE_NETWORK_ERROR {
		e.Code = exports.AICODE_CONNECTOR_NETWORK_ERROR
	}else if e.Code == exports.AILAB_REMOTE_REST_ERROR {
		e.Code = exports.AICODE_CONNECTOR_ERROR
	}
	return e
}
