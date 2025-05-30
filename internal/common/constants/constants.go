package constants

const (
	IDLength             = 8
	AgentDir             = "agents"
	OperatorListenerName = "operator"
	OperatorListenerID   = "00000000"
	AgentListenerName    = "agent"
	AgentListenerId      = "00000001"
	SshTimeout           = 30
	// number of maximum network clients timeout
	MaxClientTimeout = 300
	// number of maximum network awaiting clients
	MaxNetworkClients = 1000
	// number of bytes for reading from connection for protocol determination
	ConnHeaderLength = 16
	// maximum number of unwrapping
	MaxUnwrapAttempts = 8
)
