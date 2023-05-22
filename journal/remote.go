package journal

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

type JournalRemoteConfig struct {
	Protocol        string `json:"protocol"`
	KeyPath         string `json:"key_path"`
	CertificatePath string `json:"certificate_path"`
	TrustedCaPath   string `json:"trusted_ca_path"`
	Port            int    `json:"port"`
}

type JournalRemote struct {
	dir    string
	config *JournalRemoteConfig
}

func MakeRemote(dir string, config *JournalRemoteConfig) *JournalRemote {
	return &JournalRemote{
		dir:    dir,
		config: config,
	}
}

func (self *JournalRemote) makeCmd() (*exec.Cmd, error) {
	if self.config.Protocol == "https" {
		log.Printf("starting journal_remote: listening for https connections on port %v", self.config.Port)
		cmd := exec.Command("/lib/systemd/systemd-journal-remote",
			fmt.Sprintf("--listen-https=%v", self.config.Port),
			fmt.Sprintf("--key=%v", self.config.KeyPath),
			fmt.Sprintf("--cert=%v", self.config.CertificatePath),
			fmt.Sprintf("--trust=%v", self.config.TrustedCaPath),
			fmt.Sprintf("--output=%v", self.dir),
			"--split-mode=host",
			"--compress=false",
		)
		return cmd, nil
	}
	if self.config.Protocol == "http" {
		log.Printf("starting journal_remote: listening for http connections on port %v", self.config.Port)
		cmd := exec.Command("/lib/systemd/systemd-journal-remote",
			fmt.Sprintf("--listen-http=%v", self.config.Port),
			fmt.Sprintf("--output=%v", self.dir),
			"--split-mode=host",
			"--compress=false",
		)
		return cmd, nil
	}
	return nil, fmt.Errorf("invalid protocol: %v", self.config.Protocol)
}
func (self *JournalRemote) Start(done chan<- bool) error {
	if self.config == nil {
		return nil
	}
	cmd, err := self.makeCmd()
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	err = cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		e := cmd.Wait()
		log.Printf("journal-remote quit: err? %v", e)
		done <- true
	}()
	return nil
}
