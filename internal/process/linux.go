package process

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type Linux struct {
	Proc string
}

func NewLinux() Linux {
	return Linux{"/proc"}
}

func NewLinuxProc(proc string) Linux {
	return Linux{proc}
}

func (l Linux) TakeSnapshot() (*Snapshot, error) {
	entries, err := os.ReadDir(l.Proc)
	if err != nil {
		return nil, fmt.Errorf("take snapshot: %w", err)
	}

	var snapshot Snapshot
	snapshot.byPID = make(map[int]Process, len(entries))
	snapshot.children = make(map[int][]Process)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if _, err := strconv.ParseInt(entry.Name(), 10, 32); err != nil {
			continue
		}

		statBytes, err := os.ReadFile(filepath.Join(l.Proc, entry.Name(), "stat"))
		if err != nil {
			continue
		}

		statStr := string(statBytes)
		stat, err := parseStat(statStr)
		if err != nil {
			continue
		}

		cmdline, err := os.ReadFile(filepath.Join(l.Proc, entry.Name(), "cmdline"))
		var args []string
		if err == nil {
			args = parseCmdline(string(cmdline))
		}

		proc := Process{
			PID:  stat.pid,
			PPID: stat.ppid,
			Comm: stat.comm,
			Args: args,
		}
		snapshot.byPID[stat.pid] = proc
		snapshot.children[stat.ppid] = append(snapshot.children[stat.ppid], proc)
	}

	return &snapshot, nil
}

type statData struct {
	pid  int
	ppid int
	comm string
}

func parseStat(stat string) (statData, error) {
	cmdStart := strings.Index(stat, "(")
	cmdEnd := strings.LastIndex(stat, ")")

	if cmdStart == -1 || cmdEnd == -1 || cmdEnd < cmdStart {
		return statData{}, fmt.Errorf("invalid stat format")
	}

	pid, err := strconv.ParseInt(strings.TrimSpace(stat[0:cmdStart]), 10, 32)
	if err != nil {
		return statData{}, fmt.Errorf("parse pid: %w", err)
	}

	comm := stat[cmdStart+1 : cmdEnd]

	fields := strings.Fields(strings.TrimSpace(stat[cmdEnd+1:]))
	if len(fields) < 2 {
		return statData{}, fmt.Errorf("invalid stat fields")
	}

	ppid, err := strconv.ParseInt(fields[1], 10, 32)
	if err != nil {
		return statData{}, fmt.Errorf("parse ppid: %w", err)
	}

	return statData{
		pid:  int(pid),
		ppid: int(ppid),
		comm: comm,
	}, nil
}

func parseCmdline(cmdline string) []string {
	return slices.DeleteFunc(strings.Split(cmdline, "\x00"), func(s string) bool { return s == "" })
}
