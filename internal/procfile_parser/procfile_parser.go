package procfile_parser

import (
	"fmt"
	"os"
)

// Service type represents a single service with its data.
type Service struct {
    // The name of the service given in the procfile.
    ServiceName string

    // The underlying process once it starts.
    // it's set to nil whenever the services is stopped or terminated.
    // this field is set when the process is up and running.
    Process *os.Process

    // The command executed to start the service.
    Cmd string

    // The service is always restarted after termination, unless RunOnce is set to true.
    RunOnce bool

    // Names of this service's dependencies.
    Deps []string

    // Holds all check kinds provided with the procfile.
    Checks ServiceChecks
}

// ServiceChecks represents the checks provided in the procfile.
type ServiceChecks struct {
    // Command that runs to start the checks.
    Cmd string
    
    // Tcp ports checked if there is a process listening on them.
    TcpPorts []string

    // Udp ports checked if there is a process listening on them.
    UdpPorts []string
}

// ParseService parses the map got from yaml parser represents it using Service type.
func ParseService(serviceMap map[string]any) Service {
    service := Service{}
    for key, value := range serviceMap {
        switch key {
        case "cmd":
            service.Cmd = value.(string)
        case "run_once":
            service.RunOnce = value.(bool)
        case "deps":
            service.Deps = parseDeps(value)
        case "checks":
            checks := ServiceChecks{}
            parseCheck(value, &checks)
            service.Checks = checks
        }
    }
    return service
}

// parseDeps parses the dependencies part in the service.
func parseDeps(deps any) []string {
    var resultList []string
    depsList := deps.([]any)

    for _, dep := range depsList {
        resultList = append(resultList, dep.(string))
    }

    return resultList
}

// parseCheck parses the checks part in the service.
func parseCheck(check any, out *ServiceChecks)  {
    checkMap := check.(map[string]any)

    for key, value := range checkMap {
        switch key {
        case "cmd":
            out.Cmd = value.(string)
        case "tcp_ports":
            out.TcpPorts = parsePorts(value)
        case "udp_ports":
            out.UdpPorts = parsePorts(value)
        }
    }
}

// parsePorts parses the ports to be check in the ports part in the service.
func parsePorts(ports any) []string {
    var resultList []string
    portsList := ports.([]any)

    for _, port := range portsList {
        resultList = append(resultList, fmt.Sprint(port.(int)))
    }
    
    return resultList
}
