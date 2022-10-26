package foreman

import (
	"sync"
	"testing"

	parser "github.com/IslamWalid/foreman/internal/procparser"
)

const testProcfile = "./test-procfiles/Procfile-test"
const testBadProcfile = "./test-procfiles/Procfile-bad-test"
const testCyclicProcfile = "./test-procfiles/Procfile-cyclic-test"

func TestNew(t *testing.T) {
    t.Run("Parse existing procfile with correct syntax", func(t *testing.T) {
        want := Foreman{
        	services:      map[string]parser.Service{},
        	servicesMutex: sync.Mutex{},
        }
        sleeper := parser.Service{
        	ServiceName: "sleeper",
        	Process:     nil,
        	Cmd:         "sleep infinity",
        	RunOnce:     true,
        	Deps:        []string{"hello"},
        	Checks:      parser.ServiceChecks{
        		Cmd:      "ls",
        		TcpPorts: []string{"4759", "1865"},
        		UdpPorts: []string{"4500", "3957"},
        	},
        }
        want.services["sleeper"] = sleeper

        hello := parser.Service{
        	ServiceName: "hello",
        	Process:     nil,
        	Cmd:         `echo "hello"`,
        	RunOnce:     true,
        	Deps:        []string{},
        }
        want.services["hello"] = hello

        got, _ := New(testProcfile)
        
        assertForeman(t, got, &want)
    })

    t.Run("Run existing file with bad yml syntax", func(t *testing.T) {
        _, err := New(testBadProcfile)
        if err == nil {
            t.Error("Expcted error: yaml: unmarshal errors")
        }
    })

    t.Run("Run non-existing file", func(t *testing.T) {
        _, err := New("uknown_file")
        want := "open uknown_file: no such file or directory"
        assertError(t, err, want)
    })
}

func TestBuildDependencyGraph(t *testing.T) {
    foreman, _ := New(testProcfile)

    got := foreman.buildDependencyGraph()
    want := make(map[string][]string)
    want["sleeper"] = []string{"hello"}

    assertGraph(t, got, want)
}

func TestIsCyclic(t *testing.T) {
    t.Run("run cyclic graph", func(t *testing.T) {
        foreman, _ := New(testCyclicProcfile)
        graph := foreman.buildDependencyGraph()
        got := graph.IsCyclic()
        if !got {
            t.Error("got:true, want:false")
        }
    })

    t.Run("run acyclic graph", func(t *testing.T) {
        foreman, _ := New(testProcfile)
        graph := foreman.buildDependencyGraph()
        got := graph.IsCyclic()
        if got {
            t.Error("got:false, want:true")
        }
    })
}

func TestTopSort(t *testing.T) {
    foreman, _ := New(testProcfile)
    depGraph := foreman.buildDependencyGraph()
    got := depGraph.TopSort()
    assertTopSortResult(t, foreman, got)
}

func assertForeman(t *testing.T, got, want *Foreman) {
    t.Helper()

    for serviceName, service := range got.services {
        assertService(t, service, want.services[serviceName])
    }
}

func assertService(t *testing.T, got, want parser.Service) {
    t.Helper()

    if got.ServiceName != want.ServiceName {
        t.Errorf("got:\n%q\nwant:\n%q", got.ServiceName, want.ServiceName)
    }

    if got.Process != want.Process {
        t.Errorf("got:\n%v\nwant:\n%v", got.Process, want.Process)
    }

    if got.Cmd != want.Cmd {
        t.Errorf("got:\n%q\nwant:\n%q", got.Cmd, want.Cmd)
    }

    if got.Cmd != want.Cmd {
        t.Errorf("got:\n%q\nwant:\n%q", got.Cmd, want.Cmd)
    }

    if got.RunOnce != want.RunOnce {
        t.Errorf("got:\n%t\nwant:\n%t", got.RunOnce, want.RunOnce)
    }

    assertList(t, got.Deps, want.Deps)
}

func assertChecks(t *testing.T, got, want *parser.ServiceChecks) {
    t.Helper()

    if got.Cmd != want.Cmd {
        t.Errorf("got:\n%q\nwant:\n%q", got.Cmd, want.Cmd)
    }

    assertList(t, got.TcpPorts, want.TcpPorts)
    assertList(t, got.UdpPorts, want.UdpPorts)
}

func assertList(t *testing.T, got, want []string) {
    t.Helper()

    if len(want) != len(got) {
        t.Errorf("got:\n%v\nwant:\n%v", got, want)
    }

    n := len(want)
    for i := 0; i < n; i++ {
        if got[i] != want[i] {
            t.Errorf("got:\n%v\nwant:\n%v", got, want)
        }
    }
}

func assertError(t *testing.T, err error, want string) {
    t.Helper()

    if err == nil {
        t.Fatal("Expected Error: open uknown_file: no such file or directory")
    }

    if err.Error() != want {
        t.Errorf("got:\n%q\nwant:\n%q", err.Error(), want)
    }
}

func assertGraph(t *testing.T, got, want map[string][]string) {
    t.Helper()

    for key, value := range got {
        assertList(t, value, want[key])
    }
}

func assertTopSortResult(t *testing.T, foreman *Foreman, got []string) {
	t.Helper()

	nodesSet := make(map[string]any)
	for _, dep := range got {
		for _, depDep := range foreman.services[dep].Deps {
			if _, ok := nodesSet[depDep]; !ok {
				t.Fatalf("not expected to run %v before %v", dep, depDep)
			}
		}
		nodesSet[dep] = 1
	}
}
