package sshagent

import (
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type WrappedAgent struct {
	agents []agent.Agent
}

func NewWrappedAgent(defaultAgent agent.Agent, others []agent.Agent) *WrappedAgent {
	agents := []agent.Agent{defaultAgent}
	agents = append(agents, others...)

	return &WrappedAgent{
		agents: agents,
	}
}

func (a *WrappedAgent) List() ([]*agent.Key, error) {
	allKeys := make([]*agent.Key, 0)

	for _, agent := range a.agents {
		keys, err := agent.List()
		if err != nil {
			return nil, err
		}

		allKeys = append(allKeys, keys...)
	}

	return allKeys, nil
}

func (a *WrappedAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	return a.SignWithFlags(key, data, 0)
}

func (a *WrappedAgent) SignWithFlags(key ssh.PublicKey, data []byte, flags agent.SignatureFlags) (*ssh.Signature, error) {
	var firstError error

	for _, agent_ := range a.agents {
		var sign *ssh.Signature
		var err error
		if extendAgent, ok := agent_.(agent.ExtendedAgent); ok {
			sign, err = extendAgent.SignWithFlags(key, data, flags)
		} else {
			sign, err = agent_.Sign(key, data)
		}

		if err == nil {
			return sign, nil
		}

		if firstError == nil {
			firstError = err
		}
	}

	return nil, firstError
}

func (a *WrappedAgent) Add(key agent.AddedKey) error {
	return a.agents[0].Add(key)
}

func (a *WrappedAgent) Remove(key ssh.PublicKey) error {
	var err error

	for _, agent_ := range a.agents {
		err = agent_.Remove(key)
		if err == nil {
			return nil
		}
	}

	return err
}

func (a *WrappedAgent) RemoveAll() error {
	errs := []error{}

	for _, agent_ := range a.agents {
		err := agent_.RemoveAll()
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errs[0]
}

func (a *WrappedAgent) Lock(passphrase []byte) error {
	return a.agents[0].Lock(passphrase)
}

func (a *WrappedAgent) Unlock(passphrase []byte) error {
	return a.agents[0].Unlock(passphrase)
}

func (a *WrappedAgent) Signers() ([]ssh.Signer, error) {
	return a.agents[0].Signers()
}

func (a *WrappedAgent) Extension(extensionType string, contents []byte) ([]byte, error) {
	return nil, agent.ErrExtensionUnsupported
}
