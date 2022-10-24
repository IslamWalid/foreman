package procfile_parser

import (
	"fmt"
	"os"
)

type Service struct {
    ServiceName string
    Active bool
    Process *os.Process
    Cmd string
    RunOnce bool
    Deps []string
    Checks ServiceChecks
}

type ServiceChecks struct {
    Cmd string
    TcpPorts []string
    UdpPorts []string
}

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

func parseDeps(deps any) []string {
    var resultList []string
    depsList := deps.([]any)

    for _, dep := range depsList {
        resultList = append(resultList, dep.(string))
    }

    return resultList
}

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

func parsePorts(ports any) []string {
    var resultList []string
    portsList := ports.([]any)

    for _, port := range portsList {
        resultList = append(resultList, fmt.Sprint(port.(int)))
    }
    
    return resultList
}
