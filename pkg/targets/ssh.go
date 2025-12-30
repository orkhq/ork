package targets

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"orch.io/pkg/utils"
)

type SSHTargetConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	User string `yaml:"user"`
	Auth struct {
		Method   string `yaml:"method"`
		Password string `yaml:"password,omitempty"`
		KeyPath  string `yaml:"key_path,omitempty"`
	} `yaml:"auth"`
}

type SSHTarget struct {
	name   string
	config SSHTargetConfig

	client *ssh.Client // managed SSH client
}

func (t *SSHTarget) Name() string {
	return t.name
}

func (t *SSHTarget) Type() TargetType {
	return TargetTypeSSH
}

func (t *SSHTarget) Capabilities() Capabilities {
	return Capabilities{Exec: true, FileCopy: true}
}

func (t *SSHTarget) ValidateAndInitialize() error {
	if t.client != nil {
		return nil // already connected
	}

	if t.config.Host == "" {
		return errors.New("host required")
	}

	if t.config.User == "" {
		return errors.New("user required")
	}

	if t.config.Auth.Method == "password" && t.config.Auth.Password == "" {
		return errors.New("password authentication requires a password")
	}

	if t.config.Auth.Method == "key" && t.config.Auth.KeyPath == "" {
		return errors.New("key authentication requires a key path")
	}

	var auth []ssh.AuthMethod
	if t.config.Auth.Method == "password" {
		auth = append(auth, ssh.Password(t.config.Auth.Password))
	} else if t.config.Auth.Method == "key" {
		key, err := os.ReadFile(t.config.Auth.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	} else {
		return fmt.Errorf("unsupported auth method: %s", t.config.Auth.Method)
	}

	addr := fmt.Sprintf("%s:%d", t.config.Host, t.config.Port)
	config := &ssh.ClientConfig{
		User:            t.config.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // todo: replace with proper verification
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to dial SSH: %w", err)
	}
	t.client = client

	return nil
}

func (t *SSHTarget) Exec(ctx context.Context, req ExecCommand) (*ExecResult, error) {
	if !t.Capabilities().Exec {
		return nil, errors.New("exec not supported. target does not support Exec")
	}

	session, err := t.client.NewSession()
	if err != nil {
		return nil, err
	}

	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil {

		}
	}(session)

	if req.Stdin != nil {
		session.Stdin = req.Stdin
	}
	if req.Stdout != nil {
		session.Stdout = req.Stdout
	}
	if req.Stderr != nil {
		session.Stderr = req.Stderr
	}

	// Combine command slice into single string safely
	cmd := ""
	for _, arg := range req.Command {
		cmd += fmt.Sprintf("%q ", arg)
	}

	start := time.Now()
	err = session.Run(cmd)
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		var exitErr *ssh.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitStatus()
		}
	}

	return &ExecResult{
		ExitCode: exitCode,
		Duration: duration,
		Error:    err,
	}, nil
}

func (t *SSHTarget) CopyFile(ctx context.Context, req FileCopyRequest) (*FileCopyResult, error) {
	if !t.Capabilities().FileCopy {
		return nil, errors.New("FileCopy not supported")
	}

	sftpClient, err := sftp.NewClient(t.client)
	if err != nil {
		return nil, err
	}
	defer func(sftpClient *sftp.Client) {
		err := sftpClient.Close()
		if err != nil {

		}
	}(sftpClient)

	var srcFS, dstFS utils.FSWithPath
	if req.ToTarget {
		srcFS = utils.FSWithPath{FS: &utils.LocalFS{}, Path: req.Source}
		dstFS = utils.FSWithPath{FS: &utils.SFTPFS{SftpClient: sftpClient}, Path: req.Destination}
	} else {
		srcFS = utils.FSWithPath{FS: &utils.SFTPFS{SftpClient: sftpClient}, Path: req.Source}
		dstFS = utils.FSWithPath{FS: &utils.LocalFS{}, Path: req.Destination}
	}

	copyRes, err := utils.FSCopy(srcFS, dstFS, utils.FSCopyOptions{
		Recursive: req.Recursive,
		Overwrite: req.Overwrite,
	})

	var totalBytes int64
	var totalFiles int
	var duration time.Duration
	if err == nil {
		totalBytes = copyRes.TotalBytes
		totalFiles = copyRes.TotalFiles
		duration = copyRes.Duration
	}

	return &FileCopyResult{
		CopiedFiles: totalFiles,
		Bytes:       totalBytes,
		Duration:    duration,
		Error:       err,
	}, err
}

func (t *SSHTarget) UsesNonAmbientCredentials() (bool, []string) {
	var creds []string
	if t.config.Auth.Method == "password" {
		creds = append(creds, "SSH password")
	} else if t.config.Auth.Method == "key" {
		creds = append(creds, fmt.Sprintf("SSH key at %s", t.config.Auth.KeyPath))
	}
	return len(creds) > 0, creds
}

func (t *SSHTarget) Disconnect() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}
