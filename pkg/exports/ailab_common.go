
package exports

// Define HTTP rest service used client structures

const (
	SHOW_NORMAL         = 0
	SHOW_WITH_DELETED   = 1
	SHOW_ONLY_DELETED   = 2
)

const (
	AIStudio_labs_preset = "preset"
	AIStudio_labs_visual = "visual"
	AIStudio_labs_expert = "expert"
	AIStudio_labs_autodl = "autodl"
	// scene related labs
	AIStudio_labs_scenes = "scenes"
	AISutdio_labs_discard = "**"
)
const (
	AILAB_API_VERSION = "/api/v1"
	AILAB_ENV_ADDR   = "AILAB_ADDR"
	AILAB_ENV_LAB_ID = "AILAB_LAB_ID"
)
const (
	AILAB_RUN_STATUS_INVALID  = iota
	AILAB_RUN_STATUS_INIT
	AILAB_RUN_STATUS_STARTING  // wait for started , can kill only
	AILAB_RUN_STATUS_QUEUE
	AILAB_RUN_STATUS_SCHEDULE
	AILAB_RUN_STATUS_KILLING   // request kill job , cannot break
	AILAB_RUN_STATUS_STOPPING  // wait for stopped,  cannot break
	AILAB_RUN_STATUS_RUN
	AILAB_RUN_STATUS_COMPLETING    // saving status , pseudo end status ,cannot break
	AILAB_RUN_STATUS_SUCCESS  = 100
	AILAB_RUN_STATUS_ABORT    = 101
	AILAB_RUN_STATUS_ERROR    = 102
	AILAB_RUN_STATUS_FAIL     = 103
	AILAB_RUN_STATUS_SAVE_FAIL= 104

	// following status is internal ,no need expose to end user
	AILAB_RUN_STATUS_CLEAN      = 110  // pseudo end status , cannot break
	AILAB_RUN_STATUS_DISCARDS   = 111  // should discard any data
	AILAB_RUN_STATUS_LAB_DISCARD= 112   // should discard with lab

	AILAB_RUN_STATUS_MIN_NON_ACTIVE = AILAB_RUN_STATUS_SUCCESS
)

const (
	// has saved resource
	AILAB_RUN_FLAGS_NEED_SAVE       = 0x1
	// has refs resource
	AILAB_RUN_FLAGS_NEED_REFS       = 0x2
	// default runs will be multi-instance support
	AILAB_RUN_FLAGS_SINGLE_INSTANCE = 0x4
	// default runs will not deleted automatially when success
	AILAB_RUN_FLAGS_AUTO_DELETED = 0x8
	// support paused&resume semantics
	AILAB_RUN_FLAGS_RESUMEABLE = 0x10
	// support graceful stop semantics ?
	AILAB_RUN_FLAGS_GRACE_STOP = 0x20
	// [experimental] need run on cloud
	AILAB_RUN_FLAGS_USE_CLOUD  = 0x40

	// have prepare all resource complete
	AILAB_RUN_FLAGS_PREPARE_OK           = 0x10000
	// has delete job sched
	AILAB_RUN_FLAGS_RELEASED_JOB_SCHED   = 0x20000
	// has done for save all resources
	AILAB_RUN_FLAGS_RELEASED_SAVING      = 0x40000
	// has done for release all referenced resources
	AILAB_RUN_FLAGS_RELEASED_REFS        = 0x80000

	AILAB_RUN_FLAGS_RELEASED_DONE= AILAB_RUN_FLAGS_RELEASED_JOB_SCHED|AILAB_RUN_FLAGS_RELEASED_SAVING|AILAB_RUN_FLAGS_RELEASED_REFS
)

const (
	//AILAB_STORAGE_ROOT = "pvc://ai-labs-data"
	AILAB_DEFAULT_MOUNT= "/home/AppData"
	AILAB_OUTPUT_NAME  = "*"
	AILAB_OUTPUT_MOUNT = "_out_"
	AILAB_PIPELINE_REFER_PREFIX = "pln_"
)

const (
	AILAB_RESOURCE_TYPE_MODEL   = "model"
	AILAB_RESOURCE_TYPE_DATASET = "dataset"
	AILAB_RESOURCE_TYPE_MLRUN   = "mlrun"
	AILAB_RESOURCE_TYPE_OUTPUT  = AILAB_OUTPUT_NAME
	AILAB_RESOURCE_TYPE_CODE    = "code"
	AILAB_RESOURCE_TYPE_ENGINE  = "engine"
)

const (
	AILAB_RUN_TRAINING = "train"
	AILAB_RUN_EVALUATE = "eval"
	AILAB_RUN_SAVE     = "save"
	AILAB_RUN_VISUALIZE = "view"
	//AILAB_RUN_MINDINSIGHT = "mindinsight"
	//AILAB_RUN_TENSORBOARD = "tensorboard"
)

type SearchCond struct {
	Offset     uint
	TotalCount int64
	Next       string
	//start from 1~N
	PageNum  uint     `form:"pageNum"`
	PageSize uint     `form:"pageSize"`
	Sort     string     `form:"sort"`
	// list by app group
	Group string        `form:"group"`
	// indicate "group" list match recursively !
	MatchAll bool       `form:"matchAll"`
	// search by keyword
	SearchWord string   `form:"searchWord"`
	//enumeration for need detail return
	Detail int32        `form:"detail"`
	//enumeration for deleted item search
	Show   int32        `form:"show"`
	// filters by predefined key=value pairs
	EqualFilters map[string]string
}

type CommResponse struct{
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data interface{} `json:"data"`
}

type PagedResult struct {
	// total items matched
	Total uint `json:"total"`
	// ceil(total/pageSize) ,be zero if request pageSize is zero
	TotalPages uint `json:"totalPages,omitempty"`
	// request pageNum
	PageNum uint `json:"pageNum,omitempty"`
	// request pageSize, if zero indicate none paged querys
	PageSize uint `json:"pageSize,omitempty"`
	// used for next pagedQuery hints
	Next string     `json:"next,omitempty"`
	// used for return data
	Items interface{} `json:"items"`
}

type JobDistribute struct{
	NumPs        uint32 `json:"numPs"`
	NumPsWorker  uint32 `json:"numPsWorker"`
}


type ServiceEndpoint struct{
	Name     string `json:"name"`    // service name
	Port     uint32 `json:"port"` // service port
}


type RequestObject     = map[string]interface{}
type RespObject        = map[string]interface{}
type GObject           =   map[string]interface{}
type QueryFilterMap    =   map[string]string
type RequestTags       =   map[string]string

// define for common file informations
type FileListItem struct{
	Name      string      `json:"name"`
	//CreatedAt int64     `json:"createdAt"`
	UpdatedAt int64       `json:"createdAt"`
	Size      int64      `json:"size"`
	IsDir     bool       `json:"isDir"`
}


type ReqCreateLab struct{
	Description string                 `json:"description" `
	App       string                   `json:"app;not null"`
	Group     string                   `json:"group" `    // user defined group
	Name      string                   `json:"name"  `    // user defined name
	Classify  string                   `json:"classify,omitempty"`    // user defined classify
	Type      string                   `json:"type"`                  // system defined type preset,visual,expert,autodl,scenes
	Creator   string                   `json:"creator"`
	Tags      RequestTags              `json:"tags"`      // user defined tags
	Meta      RequestObject            `json:"meta"`
	Namespace  string                  `json:"namespace"` // system namespace this lab belong to
}

type ReqBatchCreateLab struct {
	// override per lab configuration
	Group     string `json:"group,omitempty"`
	App       string `json:"app,omitempty"`
	Creator   string `json:"creator,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	// at least 1 lab configuration must be exists
	Labs      []ReqCreateLab `json:"labs"`
}



/*
       type Resource struct{
          Type          // default will be `resourceName`
          Path          // storage path
          rpath         // pod filesystem mapped  path
          ID
          Version
          Name
          Access        // default will be 0 readonly
          SubResource: {
              "code":  "{code url}"
              "train": "{train url}"
              "infer": "{used for inference}"
          }
          // any other related fields
          ...
     }
*/

type CreateJobRequest struct{

	JobType  string                   `json:"-"`           // job type should determined by server (api interface)
	JobGroup string                   `json:"-"`           // useless now
	JobFlags uint64                   `json:"-"`           // job flags used internally
	Name     string                   `json:"name"`        // varchar[255]
	Engine   string                   `json:"engine"`
	Arch     string                   `json:"arch"`        // user expected arch , empty match all os
	Quota    interface{}              `json:"quota"`       // user specified device type and number
	// if * will allocate output path automatically and override output resource
	OutputPath  string                `json:"output"`
	Creator     string                `json:"creator"`      // user specified creator
	Description string                `json:"description"`  // user specified description
	Tags        RequestTags           `json:"tags"`         // user specified tags
	Config      RequestObject         `json:"config"`    // user specified configs
	Resource    RequestObject         `json:"resource"`  // platform related resources
	Cmd         []string              `json:"cmd"`       // user specified command line for startup

	Envs        RequestTags           `json:"envs"`
	Endpoints    [] ServiceEndpoint   `json:"endpoints"` // control job scheduler create specific endpoint when create job
}

type NotifyBackendEvents interface{
	NotifyWithEvent(evt string,lastId uint64)
	JobStatusChange(runId string)
}
func  IsRunStatusIniting(status int) bool{
	return status == AILAB_RUN_STATUS_INIT
}
func IsRunStatusSuccess(status int)bool{
	return status == AILAB_RUN_STATUS_SUCCESS
}
func IsRunStatusError(status int) bool{
	return status == AILAB_RUN_STATUS_ERROR
}
func  IsRunStatusActive(status int)      bool {
	return status > AILAB_RUN_STATUS_INVALID && status < AILAB_RUN_STATUS_MIN_NON_ACTIVE
}
func  IsRunStatusNonActive(status int)   bool{
	return status >= AILAB_RUN_STATUS_MIN_NON_ACTIVE
}
func  IsRunStatusNeedComplete(status int) bool {
	return status >= AILAB_RUN_STATUS_MIN_NON_ACTIVE && status != AILAB_RUN_STATUS_SAVE_FAIL
}
func  IsRunStatusStopping(status int)    bool{
	return status == AILAB_RUN_STATUS_KILLING || status == AILAB_RUN_STATUS_STOPPING
}
func  IsRunStatusCompleting(status int)bool{
	return status == AILAB_RUN_STATUS_COMPLETING
}
func  IsRunStatusClean(status int) bool {
	return status == AILAB_RUN_STATUS_CLEAN
}
func  IsRunStatusDiscard(status int) bool{
	return status == AILAB_RUN_STATUS_DISCARDS
}
func  IsRunStatusStarting(status int) bool{
	return status == AILAB_RUN_STATUS_STARTING
}
func  IsJobResumable(flags uint64)       bool{
	return (flags & AILAB_RUN_FLAGS_RESUMEABLE) != 0
}
func  IsJobSingleton(flags uint64)       bool{
	return (flags & AILAB_RUN_FLAGS_SINGLE_INSTANCE) != 0
}
func  IsJobNeedSave(flags uint64)    bool{
	return (flags & AILAB_RUN_FLAGS_NEED_SAVE) != 0
}
func  IsJobNeedRefs(flags uint64)    bool{
	return (flags & AILAB_RUN_FLAGS_NEED_REFS) != 0
}
func  IsJobNeedPrepare(flags uint64) bool{
	return (flags & (AILAB_RUN_FLAGS_NEED_REFS | AILAB_RUN_FLAGS_NEED_SAVE)) != 0
}

func  IsJobPrepareSuccess(flags uint64)  bool{
	return (flags & AILAB_RUN_FLAGS_PREPARE_OK)!= 0
}
func  HasJobCleanupWithJobSched(flags uint64)  bool{
     return (flags &  AILAB_RUN_FLAGS_RELEASED_JOB_SCHED) != 0
}
func  HasJobCleanupWithSaving(flags uint64)  bool{
	return (flags &  AILAB_RUN_FLAGS_RELEASED_SAVING) != 0
}
func  HasJobCleanupWithRefs(flags uint64)  bool{
	return (flags &  AILAB_RUN_FLAGS_RELEASED_REFS) != 0
}
func  IsJobRunWithCloud(flags uint64) bool {
	return (flags & AILAB_RUN_FLAGS_USE_CLOUD) != 0
}
