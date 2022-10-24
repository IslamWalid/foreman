package foreman

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IslamWalid/foreman/internal/depgraph"
	parser "github.com/IslamWalid/foreman/internal/procfile_parser"
	"gopkg.in/yaml.v3"
)

const checkInterval = 500 * time.Millisecond


type Foreman struct {
    services map[string]parser.Service
    servicesMutex sync.Mutex
}

// Parse and create a new foreman object.
// it returns error if the file path is wrong or not in yml format.
func New(procfilePath string) (*Foreman, error) {
    foreman := &Foreman{
    	services:      make(map[string]parser.Service),
    	servicesMutex: sync.Mutex{},
    }

    procfileData, err := os.ReadFile(procfilePath)
    if err != nil {
        return nil, err
    }

    procfileMap := map[string]map[string]any{}
    err = yaml.Unmarshal(procfileData, procfileMap)
    if err != nil {
        return nil, err
    }

    for key, value := range procfileMap {
        service := parser.ParseService(value)
        service.ServiceName = key
        foreman.services[key] = service
    }

    return foreman, nil
}

// Start all the services and resolve their dependencies.
func (f *Foreman) Start() error {
    var wg sync.WaitGroup
    depGraph := f.buildDependencyGraph()

    if depGraph.IsCyclic() {
        errMsg := "Cyclic dependency detected"
        return errors.New(errMsg)
    }

    startList := depGraph.TopSort()

    for _, serviceName := range startList {
        wg.Add(1)
        go func(serviceName string) {
            defer wg.Done()
            f.startService(serviceName)
        }(serviceName)
    }

    go func () {
        sigChan := make(chan os.Signal)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        <-sigChan
        f.Exit(1)
    }()

    wg.Wait()

    return nil
}

// Build graph out of services dependencies.
func (f *Foreman) buildDependencyGraph() depgraph.DepGraph {
    graph := depgraph.DepGraph{}

    for serviceName, service := range f.services {
        graph[serviceName] = service.Deps
    }

    return graph
}

func (f *Foreman) startService(serviceName string) {
    for {
        f.servicesMutex.Lock()
        service := f.services[serviceName]
        f.servicesMutex.Unlock()

        serviceExec := exec.Command("bash", "-c", service.Cmd)
        serviceExec.SysProcAttr = &syscall.SysProcAttr{
        	Setpgid: true,
        	Pgid:    0,
        }
        serviceExec.Start()
		fmt.Printf("process %s has been started\n", serviceName)

        service.Process = serviceExec.Process
        service.Active = true

        f.servicesMutex.Lock()
        f.services[serviceName] = service
        f.servicesMutex.Unlock()

        go f.checker(serviceName)
        serviceExec.Wait()
		fmt.Printf("process %s exited with %s\n", serviceName, serviceExec.ProcessState.String())
        if service.RunOnce {
            break
        }
    }
}

// Perform the checks needed on a specific pid.
func (f *Foreman) checker(serviceName string) {
    service := f.services[serviceName]
    ticker := time.NewTicker(checkInterval)
    for {
        <-ticker.C

        err := syscall.Kill(service.Process.Pid, 0)
        if err != nil {
            return
        }

        err = f.checkDeps(serviceName)
        if err != nil {
            syscall.Kill(-service.Process.Pid, syscall.SIGINT)
            fmt.Printf("checking dependencies for %s failed, services has been restarted\n", serviceName)
        }

        err = f.checkCmd(serviceName)
        if err != nil {
            syscall.Kill(-service.Process.Pid, syscall.SIGINT)
            fmt.Printf("checking process for %s failed, services has been restarted\n", serviceName)
        }

        err = f.checkPorts(serviceName, "tcp")
        if err != nil {
            syscall.Kill(-service.Process.Pid, syscall.SIGINT)
            fmt.Printf("checking listening tcp ports for %s failed, services has been restarted\n", serviceName)
        }

        err = f.checkPorts(serviceName, "udp")
        if err != nil {
            syscall.Kill(-service.Process.Pid, syscall.SIGINT)
            fmt.Printf("checking listening udp ports for %s failed, services has been restarted\n", serviceName)
        }
    }
}

func (f *Foreman) checkDeps(serviceName string) error {
    service := f.services[serviceName]

    for _, depName := range service.Deps {
        depService := f.services[depName]
        if !depService.Active {
            return errors.New("Broken dependency")
        }
    }

    return nil
}

// Perform the command in the checks.
func (f *Foreman) checkCmd(serviceName string) error {
    service := f.services[serviceName]
    checkExec := exec.Command("bash", "-c", service.Checks.Cmd)
    err := checkExec.Run()
    if err != nil {
        return err
    }
    return nil
}

// Checks all ports in the checks.
func (f *Foreman) checkPorts(serviceName, portType string) error {
    var ports []string
    service := f.services[serviceName]
    switch portType {
    case "tcp":
        ports = service.Checks.TcpPorts
    case "udp":
        ports = service.Checks.UdpPorts
    }

    for _, port := range ports {
        cmd := fmt.Sprintf("netstat -lnptu | grep %s | grep %s -m 1 | awk '{print $7}'", portType, port)
        out, _ := exec.Command("bash", "-c", cmd).Output()
        pid, err := strconv.Atoi(strings.Split(string(out), "/")[0])
        if err != nil || pid != service.Process.Pid {
            return err
        }
    }

    return nil
}

func (f *Foreman) Exit(exitStatus int) {
    f.servicesMutex.Lock()
    for _, service := range f.services {
        syscall.Kill(-service.Process.Pid, syscall.SIGINT)
    }
    f.servicesMutex.Unlock()
    os.Exit(exitStatus)
}
