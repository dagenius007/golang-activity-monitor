//go:build darwin
// +build darwin

package processes

import (
	"log"
	"os/exec"
	"strconv"
	"strings"

	. "binalyze-test/types"

	// . "binalyze-test/types"

	"golang.org/x/sys/unix"
)

func getPids() ([]int32, error) {
	pids, err := unix.SysctlKinfoProcSlice("kern.proc.all")

	pidSlice := make([]int32, 0)

	for _, pid := range pids {
		pidSlice = append(pidSlice, pid.Proc.P_pid)
	}

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return pidSlice, nil
}

func statInfo(p *Process, pid int32) (*Process, error) {
	args := "-o user=,pcpu=,%mem=,rss=,vsz=,stat=,etime=,time=,comm=,nice="

	stdout, err := exec.Command("ps", args, strconv.Itoa(int(pid))).Output()
	if err != nil {
		return p, err
	}

	stats := strings.Fields(string(stdout))

	p.PID = pid

	p.User = stats[0]

	cpuUsage, err := strconv.ParseFloat(stats[1], 32)
	if err != nil {
		return p, err
	}

	p.CpuUsage = float64(cpuUsage)

	memoryPercentageUsage, err := strconv.ParseFloat(stats[2], 32)
	if err != nil {
		return p, err
	}

	p.MemoryPercentageUsage = float64(memoryPercentageUsage)

	// I am using a larger variable because values come kb which could be large
	residentMemorySize, err := strconv.ParseUint(stats[3], 10, 64)
	if err != nil {
		return p, err
	}

	// Convert to MB
	p.ResidentMemorySize = int64(residentMemorySize / 1000)

	virtualMemorySize, err := strconv.ParseUint(stats[4], 10, 64)
	if err != nil {
		return p, err
	}

	// Convert to MB
	p.VirtualMemorySize = int64(virtualMemorySize / 1000)

	p.State = formatState(stats[5])

	p.TotalTime = stats[6]

	p.CpuTime = stats[7]

	p.Command = stats[8]

	priority, err := strconv.Atoi(stats[9])
	if err != nil {
		return p, err
	}

	p.Priority = guagePriority(priority)

	return p, nil
}
