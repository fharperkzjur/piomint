
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
	AISutdio_labs_discard = "**"
)
const (
	RUN_STATUS_INIT     = iota
	RUN_STATUS_STARTING  // wait for started
	RUN_STATUS_QUEUE
	RUN_STATUS_SCHEDULE
	RUN_STATUS_RUN
	RUN_STATUS_SAVING    // saving status
	RUN_STATUS_KILLING   // request kill job
	RUN_STATUS_STOPPING  // wait for stopped
	RUN_STATUS_FAILED  = 100
	RUN_STATUS_ERROR   = 101
	RUN_STATUS_ABORT   = 102
	RUN_STATUS_SUCCESS = 103
    // wait for cleaning all storage files then delete from db
	RUN_STATUS_CLEANING = 110
)

const (
	// default runs will be multi-instance support
	RUN_FLAGS_SINGLE_INSTANCE = 0x1
	// default runs will not deleted automatially when success
	RUN_FLAGS_AUTO_DELETED = 0x2
	// support paused&resume semantics
	RUN_FLAGS_RESUMEABLE = 0x4
	// support graceful stop semantics ?
	RUN_FLAGS_GRACE_STOP= 0x8
)

const (
	AILAB_STORAGE_ROOT = "pvc://ai-labs-data"
)

const (
	AILAB_RUN_TRAINING = "train"
	AILAB_RUN_EVALUATE = "eval"
	AILAB_RUN_SAVE     = "save"
	AILAB_RUN_VISUALIZE = "visual"
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
	Show   int32          `form:"show"`
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
	TotalPages uint `json:"totalPages"`
	// request pageNum
	PageNum uint `json:"pageNum"`
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

type JobQuota struct{
	DeviceType   string `json:"deviceType"`
	DeviceNum    uint32 `json:"deviceNum"`
	Distribute  *JobDistribute `json:"dist,omitempty"`  // if not null will use it as distributation parameters
}

type PodEndpoint struct{
	Name     string `json:"name"`    // service name
	Port     uint32 `json:"podPort"` // service port
}



type RequestObject map[string]interface{}
type RespObject    map[string]interface{}
type GObject           =   map[string]interface{}
type QueryFilterMap    =   map[string]string
type RequestTags           =   map[string]string

// define for common file informations
type FileListItem struct{
	Name      string      `json:"name"`
	//CreatedAt int64     `json:"createdAt"`
	UpdatedAt int64       `json:"createdAt"`
	Size      int64      `json:"size"`
	IsDir     int8       `json:"isDir"`
}

type MountPoint struct {
	// example:
	//    hostpath - file:///hostpath
	//    pvc      - pvc://pvc-name/subpath
	Path            string             `json:"path"`
	ContainerPath   string             `json:"containerPath"`
	ReadOnly        bool               `json:"readOnly"`
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
          // default will be `resourceName`
          Type
          Path
          ID
          Name
          Version
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
	Quota    JobQuota                 `json:"quota"`       // user specified device type and number
	// if * will allocate output path automatically and override output resource
	OutputPath  string                `json:"output"`
	Creator     string                `json:"creator"`      // user specified creator
	Description string                `json:"description"`  // user specified description
	Tags        map[string]string     `json:"tags"`         // user specified tags
	Config      map[string]interface{}   `json:"config"`    // user specified configs
	Resource    map[string]interface{}   `json:"resource"`  // platform related resources
	Cmd         string                   `json:"cmd"`       // user specified command line for startup

	Envs        map[string]string        `json:"envs"`
	Endpoints       [] PodEndpoint       `json:"endpoints"` // control job scheduler create specific endpoint when create job
}

type NotifyBackendEvents interface{
	NotifyWithEvent(evt string)
}

