
package models

import (
	"encoding/base64"
	"github.com/apulis/bmod/ai-lab-backend/pkg/exports"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)
// represent a code repository connector
type Code struct{
	RepoId    string         `json:"repoId"  gorm:"primary_key"`
	CreatedAt UnixTime       `json:"createdAt"`
	UpdatedAt UnixTime       `json:"updatedAt,omitempty"`
	Description string       `json:"description,omitempty"`
	Connector string          // predefined connector type
	Creator   string          `json:"creator" gorm:"type:varchar(255)"`
	AccessKey string          `json:"accessKey"`
	SecretKey string          `json:"secretKey,omitempty"`
	HttpUrl   string          `json:"http_url"`
	SshUrl    string          `json:"ssh_url"`
	Meta      *JsonMetaData   `json:"meta,omitempty"`
	Status    string          `json:"status"`
	Own       int             `json:"own"`
	BindCreated UnixTime      `json:"bindCreated,omitempty" gorm:"-"`
}

type CodeBind struct {
	Bind      string   `json:"bind"   gorm:"primary_key" `
    RepoId    string   `json:"repoId" gorm:"primary_key;index" `
	CreatedAt UnixTime `json:"createdAt"`
}

type BasicRepoInfo struct{
	RepoId    string
	Connector string
	Creator   string
	Own       int
	Status    string
}

const (
	list_repos_fields = "codes.repo_id,codes.created_at,codes.updated_at,code_binds.created_at as bind_created ,description,connector,creator,http_url,ssh_url,status,own"
	select_repo_basic_info = "repo_id,connector,creator,own,status"
)

func ListAppRepos(bind string,req*exports.SearchCond) (interface{},APIError){

	return makePagedQuery(db.Model(&CodeBind{}).Joins("left join codes using(repo_id)").
		Select(list_repos_fields).Where("bind=?",bind),req,&[]Code{})
}

func QueryAppRepoDetail(repoId string) (*Code,APIError) {
	code := &Code{}
	err := wrapDBQueryError(db.First(code,"repo_id = ?",repoId))
	if err != nil {
		code = nil
	}
	return code,err
}

func CreateAppRepoBind(code*Code,bind string,isMulti bool) APIError {
	return execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {

		if !isMulti {//check singleton binding
			if appBind,err := getBindRepoInfo(tx,bind);err == nil {
				if code.RepoId == appBind.RepoId {
					return exports.RaiseAPIError(exports.AICODE_CONNECTOR_ALREADY_EXISTS,
						"app singleton repo bind already exists !!!")
				}
				code.RepoId=appBind.RepoId
				return exports.RaiseAPIError(exports.AICODE_CONNECTOR_ERROR,
					"app singleton repo another bind exists !!!")
			}else if err.Errno() != exports.AILAB_NOT_FOUND {
				return err
			}
		}
        var err APIError
		if len(code.RepoId) == 0 {//create repo first
			uuid_buf := uuid.New()
			code.RepoId= base64.RawURLEncoding.EncodeToString(uuid_buf[:])
			err=wrapDBUpdateError(tx.Create(code),1)
		}else{//check repo exists
			if repo ,err := getBasicRepoInfo(tx,code.RepoId);err != nil {
				return err
			}else{// check repo compatible
				if repo.Status != exports.AICODE_REPO_STATUS_READY{
					return exports.RaiseAPIError(exports.AICODE_CONNECTOR_INVALID_STATUS,"bind exists repo invalid status !!!")
				}
				if repo.Own != code.Own {
					return exports.RaiseAPIError(exports.AICODE_CONNECTOR_ERROR,"conflict bind own flags !!!")
				}
				if repo.Connector != code.Connector {
					return exports.RaiseAPIError(exports.AICODE_CONNECTOR_ERROR,"conflict bind connector names !!!")
				}
			}
		}
		if err == nil {
			err = wrapDBUpdateError(tx.Create(&CodeBind{
				Bind:      bind,
				RepoId:    code.RepoId,
			}),1)
			if err1,ok := err.(*exports.APIException);ok && err1.Code == exports.AILAB_DB_DUPLICATE{
				err1.Code=exports.AICODE_CONNECTOR_ALREADY_EXISTS
				err1.Msg="app multi repo bind already exists !!!"
			}
		}
		return err
	})
}

func DeleteAppRepoBind(app,repoId string) (interface{},APIError ){

	counts := 0
	err    := execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {

		var inst *gorm.DB
		if len(repoId) == 0 {
			inst = tx.Model(&CodeBind{}).Where("bind=?",app)
		}else{
            inst = tx.Model(&CodeBind{}).Where("bind=? and repo_id=?",app,repoId)
		}
		repoIdSets := []string{}
		if err:= wrapDBQueryError(inst.Select("repo_id").Clauses(clause.Locking{Strength: "UPDATE"}).Find(&repoIdSets));err != nil {
			return err
		}
		for _,repoId = range(repoIdSets) {

			if err := wrapDBUpdateError(tx.Delete(&CodeBind{},"bind=? and repo_id=?",app,repoId),1);err != nil{
				return err
			}
			if no_binds,err := checkRepoNonBinds(tx,repoId);err != nil{
				return err
			}else if no_binds{//here need to delete repo also
				if err= deleteRepo(tx,repoId,events);err != nil {
					return err
				}
			}
			counts++
		}
		return nil
	})
	return counts,err
}

func RollBackAllInitRepos() (int,APIError){
	counts := 0
	err    := execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {
		repoIdSets := []string{}
		if err:= wrapDBQueryError(tx.Table("codes").Select("repo_id").Where("status=?",exports.AICODE_REPO_STATUS_INIT).
			Clauses(clause.Locking{Strength: "UPDATE"}).Find(&repoIdSets));err != nil {
			return err
		}
		for _,repoId := range(repoIdSets){
			if err := wrapDBUpdateError(tx.Delete(&CodeBind{},"repo_id=?",repoId),0);err != nil{
				return err
			}
			if err := deleteRepo(tx,repoId,events);err != nil {
				return err
			}
			counts++
		}
		return nil
	})
	return counts,err
}

func DeleteRepoActually(repoId string) APIError {
	return wrapDBUpdateError(db.Delete(&Code{},"repo_id=? and status=?",repoId,exports.AICODE_REPO_STATUS_DELETE),0)
}

func CompleteInitRepo(code *Code,success bool) APIError {
	return    execDBTransaction(func(tx *gorm.DB,events EventsTrack) APIError {
		if success {
			return wrapDBUpdateError(tx.Model(code).UpdateColumns(map[string]interface{}{
				"http_url":code.HttpUrl,
				"ssh_url" :code.SshUrl,
				"access_key":code.AccessKey,
				"secret_key":code.SecretKey,
				"status":exports.AICODE_REPO_STATUS_READY,
			}),1)
		}else{
			return deleteRepo(tx,code.RepoId,events)
		}
	})
}

func getBasicRepoInfo(tx * gorm.DB, repoId string) (repo *BasicRepoInfo,err APIError) {
	repo = &BasicRepoInfo{}
	err = wrapDBQueryError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&Code{}).
		Select(select_repo_basic_info).First(repo,"repo_id=?",repoId))
	return
}
func getBindRepoInfo(tx * gorm.DB, bind string) (app *CodeBind,err APIError) {
	app = &CodeBind{}
	err = wrapDBQueryError(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(app,"bind=?",bind))
	return
}
func checkRepoNonBinds(tx*gorm.DB,repoId string) (bool,APIError){
	if err := wrapDBQueryError(tx.First(&CodeBind{},"repo_id=?",repoId));err == nil {
		return false,nil
	}else if err.Errno() == exports.AILAB_NOT_FOUND {
		return true,nil
	}else {
		return false,err
	}
}
func deleteRepo(tx*gorm.DB,repoId string,track EventsTrack) APIError{
	if err := wrapDBUpdateError(tx.Model(&Code{}).Where("repo_id=?",repoId).
		UpdateColumn("status",exports.AICODE_REPO_STATUS_DELETE),1);err != nil{
		return err
	}else{
		return LogBackendEvent(tx,Evt_delete_repo,repoId,nil,track)
	}
}
