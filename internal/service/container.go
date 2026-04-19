package service

import (
	"fmt"
	"io"
)

// ContainerRunner provides container-level operations. The infra layer
// provides the real implementation; tests use a fake.
type ContainerRunner interface {
	FindContainer(projectName, serviceName string) (string, error)
	Exec(containerName string, cmd []string, tty bool, user string) error
	Logs(containerName string, follow bool, tail string) (io.ReadCloser, error)
}

// ContainerService handles interaction with running containers:
// shell access, command execution, and log streaming.
// It looks up the target container via FindContainer, then delegates
// to the ContainerRunner for the actual operation.
type ContainerService struct {
	runner ContainerRunner
}

// NewContainerService creates a ContainerService.
func NewContainerService(runner ContainerRunner) *ContainerService {
	return &ContainerService{runner: runner}
}

// Shell opens an interactive shell in the container for the given project
// and service. The cmd parameter is the shell command and any args
// (e.g. []string{"bash", "-l"}). When user is non-empty, the shell runs as
// that user; otherwise the image's default user is used.
func (s *ContainerService) Shell(project, service string, cmd []string, user string) error {
	containerID, err := s.runner.FindContainer(project, service)
	if err != nil {
		return fmt.Errorf("finding container: %w", err)
	}

	if err := s.runner.Exec(containerID, cmd, true, user); err != nil {
		return fmt.Errorf("shell exec: %w", err)
	}

	return nil
}

// Exec runs a command (without TTY) in the container for the given project
// and service. When user is non-empty, the command runs as that user.
func (s *ContainerService) Exec(project, service string, cmd []string, user string) error {
	containerID, err := s.runner.FindContainer(project, service)
	if err != nil {
		return fmt.Errorf("finding container: %w", err)
	}

	if err := s.runner.Exec(containerID, cmd, false, user); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

// Logs returns a reader streaming the container's logs. The caller is
// responsible for closing the returned reader.
func (s *ContainerService) Logs(project, service string, follow bool, tail string) (io.ReadCloser, error) {
	containerID, err := s.runner.FindContainer(project, service)
	if err != nil {
		return nil, fmt.Errorf("finding container: %w", err)
	}

	rc, err := s.runner.Logs(containerID, follow, tail)
	if err != nil {
		return nil, fmt.Errorf("streaming logs: %w", err)
	}

	return rc, nil
}
