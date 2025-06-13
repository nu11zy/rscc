package constants

const (
	IDLength             = 8
	AgentDir             = "agents"
	OperatorListenerName = "operator"
	OperatorListenerID   = "00000000"
	AgentListenerName    = "agent"
	AgentListenerID      = "00000001"
	SshTimeout           = 30
	MaxUnwrapConnections = 1000
	MaxUnwrapDepth       = 8
)

var Subsystems = []string{"kill", "sftp", "pscan", "pfwd", "executeassembly"}
