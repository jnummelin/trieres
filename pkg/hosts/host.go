package hosts

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type RemoteHost interface {
	Connect() error
	Disconnect() error
}

// Config for host
type Config struct {
	Address    string
	User       string
	SSHPort    int
	SSHKeyPath string
	Role       string
	ExtraArgs  []string
}

// Host describes connectable host
type Host struct {
	Config
	sshClient *ssh.Client
}

type Hosts []*Host

// Connect to the host
func (h *Host) Connect() error {
	key, err := ioutil.ReadFile(h.SSHKeyPath)
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return err
	}
	config := ssh.ClientConfig{
		User: h.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	address := fmt.Sprintf("%s:%d", h.Address, h.SSHPort)
	client, err := ssh.Dial("tcp", address, &config)
	if err != nil {
		return err
	}
	h.sshClient = client

	return nil
}

// Exec a command on the host
func (h *Host) Exec(cmd string) error {
	session, err := h.sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		fmt.Println("FOOO")
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	if err := session.Start(cmd); err != nil {
		return err
	}

	multiReader := io.MultiReader(stdout, stderr)
	outputScanner := bufio.NewScanner(multiReader)

	for outputScanner.Scan() {
		logrus.Debugf("%s:  %s", h.Address, outputScanner.Text())
	}
	if err := outputScanner.Err(); err != nil {
		logrus.Errorf("%s:  %s", h.Address, err.Error())
	}

	return nil
}

// Disconnect from the host
func (h *Host) Disconnect() error {
	if h.sshClient == nil {
		return nil
	}

	return h.sshClient.Close()
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfig Config
	homeDir, _ := homedir.Dir()
	raw := rawConfig{
		Address:    "127.0.0.1",
		User:       "root",
		SSHKeyPath: path.Join(homeDir, ".ssh", "id_rsa"),
		SSHPort:    22,
		Role:       "worker",
	}

	if err := unmarshal(&raw); err != nil {
		return err
	}

	*c = Config(raw)
	return nil
}
