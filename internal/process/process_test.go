package process

import (
	"slices"
	"testing"
)

func TestSnapshotFind(t *testing.T) {
	snapshot := testSnapshot([]Process{
		{PID: 10, PPID: 1, Comm: "bash"},
		{PID: 11, PPID: 10, Comm: "codex"},
	})

	got, ok := snapshot.Find(11)
	if !ok {
		t.Fatal("Find(11) did not find process")
	}

	want := Process{PID: 11, PPID: 10, Comm: "codex"}
	if got.PID != want.PID || got.PPID != want.PPID || got.Comm != want.Comm {
		t.Fatalf("Find(11) = %+v, want %+v", got, want)
	}

	if _, ok := snapshot.Find(99); ok {
		t.Fatal("Find(99) found unexpected process")
	}
}

func TestSnapshotChildrenOf(t *testing.T) {
	snapshot := testSnapshot([]Process{
		{PID: 10, PPID: 1, Comm: "bash"},
		{PID: 11, PPID: 10, Comm: "devenv"},
		{PID: 12, PPID: 11, Comm: "codex"},
		{PID: 13, PPID: 10, Comm: "sleep"},
	})

	got := pids(snapshot.ChildrenOf(10))
	want := []int{11, 13}
	if !slices.Equal(got, want) {
		t.Fatalf("ChildrenOf(10) = %+v, want %+v", got, want)
	}

	if got := snapshot.ChildrenOf(12); len(got) != 0 {
		t.Fatalf("ChildrenOf(12) = %+v, want empty", got)
	}
}

func TestSnapshotDescendantsOf(t *testing.T) {
	snapshot := testSnapshot([]Process{
		{PID: 1, PPID: 0, Comm: "init"},
		{PID: 10, PPID: 1, Comm: "bash"},
		{PID: 11, PPID: 10, Comm: "devenv"},
		{PID: 12, PPID: 11, Comm: "codex"},
		{PID: 13, PPID: 10, Comm: "sleep"},
		{PID: 20, PPID: 1, Comm: "other"},
	})

	got := pids(snapshot.DescendantsOf(10))
	want := []int{11, 13, 12}
	if !slices.Equal(got, want) {
		t.Fatalf("DescendantsOf(10) = %+v, want %+v", got, want)
	}

	got = pids(snapshot.DescendantsOf(11))
	want = []int{12}
	if !slices.Equal(got, want) {
		t.Fatalf("DescendantsOf(11) = %+v, want %+v", got, want)
	}

	if got := snapshot.DescendantsOf(12); len(got) != 0 {
		t.Fatalf("DescendantsOf(12) = %+v, want empty", got)
	}

	if got := snapshot.DescendantsOf(99); len(got) != 0 {
		t.Fatalf("DescendantsOf(99) = %+v, want empty", got)
	}
}

func testSnapshot(processes []Process) *Snapshot {
	snapshot := &Snapshot{
		byPID:    make(map[int]Process, len(processes)),
		children: make(map[int][]Process),
	}

	for _, proc := range processes {
		snapshot.byPID[proc.PID] = proc
		snapshot.children[proc.PPID] = append(snapshot.children[proc.PPID], proc)
	}

	return snapshot
}

func pids(processes []Process) []int {
	pids := make([]int, 0, len(processes))
	for _, proc := range processes {
		pids = append(pids, proc.PID)
	}

	return pids
}
