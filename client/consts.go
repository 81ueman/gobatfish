package client

// BfConsts contains constants derived from BfConsts.java in batfish-common.
// It must be kept in sync with the Java version.
const (
	ArgAnswerJSONPath               = "answerjsonpath"
	ArgBlockNames                   = "blocknames"
	ArgContainerDir                 = "containerdir"
	ArgDeltaTestrig                 = "deltatestrig"
	ArgDiffActive                   = "diffactive"
	ArgDifferential                 = "differential"
	ArgHaltOnConvertError           = "haltonconverterror"
	ArgHaltOnParseError             = "haltonparseerror"
	ArgIgnoreFilesWithStrings       = "ignorefileswithstrings"
	ArgLogFile                      = "logfile"
	ArgLogLevel                     = "loglevel"
	ArgPedanticAsError              = "pedanticerror"
	ArgPedanticSuppress             = "pedanticsuppress"
	ArgPluginDirs                   = "plugindirs"
	ArgPrettyPrintAnswer            = "ppa"
	ArgQuestionName                 = "questionname"
	ArgRedFlagAsError               = "redflagerror"
	ArgRedFlagSuppress              = "redflagsuppress"
	ArgSynthesizeJSONTopology       = "synthesizejsontopology"
	ArgSynthesizeTopology           = "synthesizetopology"
	ArgTaskPlugin                   = "taskplugin"
	ArgTestrig                      = "testrig"
	ArgUnimplementedAsError         = "unimplementederror"
	ArgUnimplementedSuppress        = "unimplementedsuppress"
	ArgUnrecognizedAsRedFlag        = "urf"
	ArgUsePrecomputedAdvertisements = "useprecomputedadvertisements"
	ArgUsePrecomputedIbgpNeighbors  = "useprecomputedibgpneighbors"
	ArgUsePrecomputedRoutes         = "useprecomputedroutes"
	ArgVerboseParse                 = "verboseparse"

	CommandAnswer                 = "answer"
	CommandDumpDP                 = "dp"
	CommandInitInfo               = "initinfo"
	CommandParseVendorIndependent = "si"
	CommandParseVendorSpecific    = "sv"
	CommandQuery                  = "query"
	CommandReport                 = "report"
	CommandValidateEnvironment    = "venv"
	CommandWriteAdvertisements    = "writeadvertisements"
	CommandWriteIbgpNeighbors     = "writeibgpneighbors"
	CommandWriteRoutes            = "writeroutes"

	SvcBaseRSC = "/batfishservice"
	SvcPort    = 9999

	SuffixAnswerJSONFile = ".json"
	SuffixLogFile        = ".log"
)

// WorkStatusCode is the status of a Batfish work item.
type WorkStatusCode string

// Work status codes, mirroring pybatfish.client.consts.WorkStatusCode.
const (
	WorkAssigned             WorkStatusCode = "ASSIGNED"
	WorkAssignmentError      WorkStatusCode = "ASSIGNMENTERROR"
	WorkBlocked              WorkStatusCode = "BLOCKED"
	WorkCheckingStatus       WorkStatusCode = "CHECKINGSTATUS"
	WorkRequeueFailure       WorkStatusCode = "REQUEUEFAILURE"
	WorkTerminatedAbnormally WorkStatusCode = "TERMINATEDABNORMALLY"
	WorkTerminatedByUser     WorkStatusCode = "TERMINATEDBYUSER"
	WorkTerminatedNormally   WorkStatusCode = "TERMINATEDNORMALLY"
	WorkTryingToAssign       WorkStatusCode = "TRYINGTOASSIGN"
	WorkUnassigned           WorkStatusCode = "UNASSIGNED"
)

// IsTerminated reports whether the work status is a terminal state.
func (w WorkStatusCode) IsTerminated() bool {
	switch w {
	case WorkAssignmentError, WorkRequeueFailure, WorkTerminatedAbnormally, WorkTerminatedByUser, WorkTerminatedNormally:
		return true
	default:
		return false
	}
}

// CoordConsts contains constants derived from CoordConsts.java in batfish-common.
const (
	DefaultAPIKey = "00000000000000000000000000000000"

	SvcCfgWorkMgr2       = "/v2"
	SvcCfgWorkV2Port     = 9996
	SvcCfgWorkPort       = 9997
	SvcCfgWorkSSLDisable = true

	HTTPHeaderBatfishAPIKey  = "X-Batfish-Apikey"
	HTTPHeaderBatfishVersion = "X-Batfish-Version"

	KeyAPIVersion = "api_version"

	SvcKeyNetworkName           = "networkname"
	SvcKeyQuestionName          = "questionname"
	SvcKeySnapshotName          = "snapshotname"
	SvcKeyReferenceSnapshotName = "referencesnapshotname"
	SvcKeyTaskStatus            = "taskstatus"
	SvcKeyWorkStatus            = "workstatus"
	SvcKeyWorkList              = "worklist"
	SvcKeySuggestions           = "suggestions"
	SvcKeyZipFile               = "zipfile"
	SvcKeyFile                  = "file"
	SvcKeyAPIKey                = "apikey"
	SvcKeyResult                = "result"
	SvcKeyAnswer                = "answer"
)

// CoordConstsV2 contains constants derived from CoordConstsV2.java.
const (
	HTTPHeaderBatfishAPIKey2  = "X-Batfish-Apikey"
	HTTPHeaderBatfishVersion2 = "X-Batfish-Version"

	QPKey            = "key"
	QPVerbose        = "verbose"
	QPMaxSuggestions = "maxsuggestions"
	QPName           = "name"
	QPQuery          = "query"

	RSCAnswer            = "answer"
	RSCAutoComplete      = "autocomplete"
	RSCFork              = "fork"
	RSCInferredNodeRoles = "inferred_node_roles"
	RSCInput             = "input"
	RSCNetworks          = "networks"
	RSCNodeRoles         = "noderoles"
	RSCObjects           = "objects"
	RSCQuestions         = "questions"
	RSCQuestionTemplates = "question_templates"
	RSCReferenceLibrary  = "referencelibrary"
	RSCSettings          = "settings"
	RSCSnapshots         = "snapshots"
	RSCWork              = "work"
	RSCWorkLog           = "worklog"

	PropTask           = "task"
	PropWorkStatusCode = "workstatuscode"
)
