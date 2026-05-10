package process

type Process struct {
	PID  int
	PPID int
	Comm string
	Args []string
}

type Snapshot struct {
	byPID    map[int]Process
	children map[int][]Process
}

type ProcessMonitor interface {
	TakeSnapshot() (*Snapshot, error)
}

func (s *Snapshot) Find(pid int) (Process, bool) {
	val, ok := s.byPID[pid]
	return val, ok
}

func (s *Snapshot) ChildrenOf(pid int) []Process {
	return s.children[pid]
}

func (s *Snapshot) DescendantsOf(pid int) []Process {
	descendants := make([]Process, 0)
	queue := []int{pid}

	for i := 0; i < len(queue); i++ {
		pid := queue[i]
		children := s.children[pid]
		descendants = append(descendants, children...)

		for _, child := range children {
			queue = append(queue, child.PID)
		}
	}

	return descendants
}
