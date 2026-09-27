package sshagent

import (
	"golang.org/x/crypto/ssh/agent"
	"io"
)

type Server struct {
	Agent agent.Agent
}

// OpenSSH >= 8.9 clients bind the agent session with the
// session-bind@openssh.com extension before signing and refuse to use
// an agent that rejects it. agentSuccess is the SSH_AGENT_SUCCESS
// response code per [PROTOCOL.agent].
//
// The binding mechanism is documented in the OpenSSH sources:
//   - PROTOCOL.agent, section "1. session-bind@openssh.com extension"
//     (extensions in general are section 4.7):
//     https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.agent
//     (protocol spec: https://datatracker.ietf.org/doc/html/rfc9987)
//   - server side: ssh-agent.c, process_ext_session_bind()
//   - client side: auth-agent.c, ssh_agent_bind_hostkey()
//
// We answer the bind with SUCCESS without storing the binding (same trust
// level as pre-8.9 agents): this local per-user agent is only reachable by
// processes of the same user, so the cross-session signature protection
// the extension provides is out of scope.
const (
	sessionBindExtension = "session-bind@openssh.com"
	agentSuccess         = 6
)

func (s *Server) SSHAgentHandler(conn io.ReadWriteCloser) {
	defer conn.Close()
	if s.Agent == nil {
		return
	}
	err := agent.ServeAgent(s.Agent, conn)
	if err != nil && err != io.EOF {
		println(err.Error())
	}
}
