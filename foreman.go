package foreman

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IslamWalid/foreman/internal/depgraph"
	parser "github.com/IslamWalid/foreman/internal/procparser"
	"gopkg.in/yaml.v3"
)

// checkInterval is period between each check performed for every running service.
const checkInterval = 500 * time.Millisecond

// Foreman provides the main methods used to start and exit services.
type Foreman struct {
	// services is a map that maps each service name with its corresponding service type.
	services map[string]parser.Service

	// servicesMutex is a mutex used to safely access the services map.
	servicesMutex sync.Mutex

	// logger is used to print log messages to stdout.
	logger *log.Logger

	// verbose specifies whether to print log messages or not.
	verbose bool
}

// Parse and create a new foreman object.
// it returns error if the file path is wrong or not in yml format.
func New(procfilePath string, verbose bool) (*Foreman, error) {
	foreman := &Foreman{
		services:      make(map[string]parser.Service),
		servicesMutex: sync.Mutex{},
		logger:        log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime),
		verbose:       verbose,
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

// vLog creates a logger for foreman verbose messages.
func (f *Foreman) vLog(msg string) {
	if f.verbose {
		f.logger.Print(msg)
	}
}

// Start resolves the dependencies between services.
// It starts the services in the appropriate order.
// returns error if dependencies cannot be resoved due to cycles in dependencies.
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

	go func() {
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

// startService starts the service with the given service name.
// It associated the service with a goroutine.
// The goroutine restarts the services whenever it terminates.
// It associates a checking procedure with the running service.
func (f *Foreman) startService(serviceName string) {
	stopCheck := make(chan bool)
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
		f.vLog(fmt.Sprintf("%s has been started\n", serviceName))

		service.Process = serviceExec.Process

		f.servicesMutex.Lock()
		f.services[serviceName] = service
		f.servicesMutex.Unlock()

		go f.checker(service, stopCheck)

		serviceExec.Wait()
		service.Process = nil
		stopCheck <- true

		f.servicesMutex.Lock()
		f.services[serviceName] = service
		f.servicesMutex.Unlock()

		f.vLog(fmt.Sprintf("%s exited with %s\n", serviceName, serviceExec.ProcessState.String()))

		if service.RunOnce {
			break
		}
	}
}

// checker runs all needed checking routines for the given services.
// The checker terminates the services upon any check failure.
// dependency check, checking command and ports check are always running
// on the specified service.
func (f *Foreman) checker(service parser.Service, stopCheck <-chan bool) {
	ticker := time.NewTicker(checkInterval)
	f.vLog(fmt.Sprintf("%s checks started\n", service.ServiceName))
	for {
		select {
		case <-stopCheck:
			f.vLog(fmt.Sprintf("%s checks stopped\n", service.ServiceName))
			return
		case <-ticker.C:
			err := f.checkDeps(service)
			if err != nil {
				syscall.Kill(-service.Process.Pid, syscall.SIGINT)
				f.vLog(fmt.Sprintf("checking dependencies for %s failed, services has been restarted\n", service.ServiceName))
			}

			err = f.checkCmd(service)
			if err != nil {
				syscall.Kill(-service.Process.Pid, syscall.SIGINT)
				f.vLog(fmt.Sprintf("checking process for %s failed, services has been restarted\n", service.ServiceName))
			}

			err = f.checkPorts(service, "tcp")
			if err != nil {
				syscall.Kill(-service.Process.Pid, syscall.SIGINT)
				f.vLog(fmt.Sprintf("checking listening tcp ports for %s failed, services has been restarted\n", service.ServiceName))
			}

			err = f.checkPorts(service, "udp")
			if err != nil {
				syscall.Kill(-service.Process.Pid, syscall.SIGINT)
				f.vLog(fmt.Sprintf("checking listening udp ports for %s failed, services has been restarted\n", service.ServiceName))
			}
		}
	}
}

// checkDeps checks that all the service dependencies are running well.
func (f *Foreman) checkDeps(service parser.Service) error {
	for _, depName := range service.Deps {
		f.servicesMutex.Lock()
		depService := f.services[depName]
		f.servicesMutex.Unlock()

		if depService.Process == nil {
			return errors.New("Broken dependency")
		}
	}

	return nil
}

// Perform the command in the checks for the given service.
func (f *Foreman) checkCmd(service parser.Service) error {
	checkExec := exec.Command("bash", "-c", service.Checks.Cmd)
	err := checkExec.Run()
	if err != nil {
		return err
	}
	return nil
}

// Checks all ports in the checks.
func (f *Foreman) checkPorts(service parser.Service, portType string) error {
	var ports []string
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

// ُExit kills all the running services and checkrs.
// exits foreman with the given exit status.
func (f *Foreman) Exit(exitStatus int) {
	f.servicesMutex.Lock()
	for _, service := range f.services {
		if service.Process != nil {
			syscall.Kill(-service.Process.Pid, syscall.SIGINT)
		}
	}
	f.servicesMutex.Unlock()
	os.Exit(exitStatus)
}
