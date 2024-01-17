package journal

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

type JournalReader struct {
	Directory    string
	AllowedUnits map[string]bool
	AllowedHosts map[string]bool
}

func MakeReader(directory string, allowedUnits []string, allowedHosts []string) *JournalReader {
	au := make(map[string]bool)
	for _, u := range allowedUnits {
		au[u] = true
	}
	ah := make(map[string]bool)
	for _, h := range allowedHosts {
		ah[h] = true
	}
	return &JournalReader{
		Directory:    directory,
		AllowedUnits: au,
		AllowedHosts: ah,
	}
}

type StreamRequest struct {
	Units        []string      `json:"units"`
	Hosts        []string      `json:"hosts"`
	RangeLines   *RangeLines   `json:"rangeLines"`
	RangePeriod  *RangePeriod  `json:"rangePeriod"`
	RangeMinutes *RangeMinutes `json:"rangeMinutes"`
	Filter       string        `json:"filter"`
}

type RangeLines struct {
	Lines  int  `json:"lines"`
	Follow bool `json:"follow"`
}

type RangePeriod struct {
	Since string `json:"since"`
	Until string `json:"until"`
}

type RangeMinutes struct {
	Minutes uint `json:"minutes"`
	Follow  bool `json:"follow"`
}

func allowed(requested []string, allowed map[string]bool) []string {
	//when allowed is empty, everythign is allowed
	if len(allowed) == 0 {
		return requested
	}
	vs := make([]string, 0)
	for _, r := range requested {
		if allowed[r] == true {
			vs = append(vs, r)
		}
	}
	if len(vs) != 0 {
		return vs
	}
	//empty valid requested values == all allowed
	for k := range allowed {
		vs = append(vs, k)
	}
	return vs
}

func Map[T any, R any](values []T, cb func(T) R) []R {
	r := make([]R, len(values), len(values))
	for _, v := range values {
		r = append(r, cb(v))
	}
	return r
}

func Filter[T any](values []T, cb func(T) bool) []T {
	r := make([]T, 0, len(values))
	for _, v := range values {
		if cb(v) {
			r = append(r, v)
		}
	}
	return r
}

func (self *JournalReader) KnownAllowedUnits() ([]string, error) {
	bytes, err := exec.Command("journalctl", "--field", "_SYSTEMD_UNIT").Output()
	if err != nil {
		return nil, err
	}

	units := Filter(strings.Split(string(bytes), "\n"), func(line string) bool {
		return line != ""
	})
	services := Filter(units, func(unit string) bool {
		return strings.HasSuffix(unit, ".service")
	})

	serviceNames := Map(services, func(service string) string {
		return strings.TrimSuffix(service, ".service")
	})

	return allowed(serviceNames, self.AllowedUnits), nil
}

func (self *JournalReader) KnownAllowedHosts() ([]string, error) {
	bytes, err := exec.Command("journalctl", "--field", "_HOSTNAME").Output()
	if err != nil {
		return nil, err
	}
	hosts := Filter(strings.Split(string(bytes), "\n"), func(line string) bool {
		return line != ""
	})
	return allowed(hosts, self.AllowedHosts), nil
}

func (self *JournalReader) Stream(c *websocket.Conn, req *StreamRequest) error {

	args := []string{
		"--merge",
		fmt.Sprintf("--directory=%s", self.Directory),
		"--output=json",
		"--all",
		"--output-fields=_HOSTNAME,_PID,_SYSTEMD_UNIT,MESSAGE",
	}

	for _, h := range allowed(req.Hosts, self.AllowedHosts) {
		args = append(args, fmt.Sprintf("_HOSTNAME=%s", h))
	}

	for _, u := range allowed(req.Units, self.AllowedUnits) {
		args = append(args, "--unit", fmt.Sprintf("%s.service", u))
	}
	if req.RangeLines == nil && req.RangePeriod == nil && req.RangeMinutes == nil {
		//default
		req.RangeMinutes = &RangeMinutes{
			Minutes: 5,
			Follow:  false,
		}
	}

	if req.Filter != "" {
		args = append(args, fmt.Sprintf("--grep=%s", req.Filter))
		args = append(args, "--case-sensitive=false")
	}

	if req.RangeLines != nil {
		args = append(args, fmt.Sprintf("--lines=%d", req.RangeLines.Lines))
		if req.RangeLines.Follow {
			args = append(args, "--follow")
		}
	} else if req.RangePeriod != nil {
		if req.RangePeriod.Since == "" && req.RangePeriod.Until == "" {
			return fmt.Errorf("either since or until must be configured")
		}
		if req.RangePeriod.Since != "" {
			args = append(args, fmt.Sprintf("--since=%s", req.RangePeriod.Since))
		}
		if req.RangePeriod.Until != "" {
			args = append(args, fmt.Sprintf("--until=%s", req.RangePeriod.Until))
		}
	} else {
		args = append(args, fmt.Sprintf("--since=-%vm", req.RangeMinutes.Minutes))
		if req.RangeMinutes.Follow {
			args = append(args, "--follow")
		}
	}

	log.Printf("args: %+v", args)
	cmd := exec.Command("journalctl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("executing journalctl: %w", err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("starting journalctl: %w", err)
	}

	var wg sync.WaitGroup
	done := make(chan bool)
	wg.Add(1)

	defer func() {
		done <- true
		close(done)
		wg.Wait()
		cmd.Wait()
	}()

	go func() {
		for {
			select {
			case <-time.After(10 * time.Second):
				//FIXME: mutex around writeMessage
				//err := c.WriteMessage(websocket.PingMessage, []byte{})
				//if err != nil {
				//	wg.Done()
				//	return
				//}
			case <-done:
				wg.Done()
				return
			}
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		message := []byte(scanner.Text())
		err := c.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			return fmt.Errorf("writing message: %w", err)
		}
	}
	return nil
}
