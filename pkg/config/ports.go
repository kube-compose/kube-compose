package config

import (
	"fmt"
	"strconv"
)

type Port struct {
	ContainerPort int32
	ExternalPort int32
	Protocol string
}

// https://docs.docker.com/compose/compose-file/compose-file-v2/
// ports:
//  - "3000"
//  - "3000-3005"
//  - "8000:8000"
//  - "9090-9091:8080-8081"
//  - "49100:22"
//  - "127.0.0.1:8001:8001"
//  - "127.0.0.1:5000-5010:5000-5010"
//  - "6060:6060/udp"
//  - "12400-12500:1240"
func parsePorts (inPorts []string) ([]Port, error) {
	n := len(inPorts)
	if n == 0 {
		return nil, nil
	}
	outPorts := make([]Port, n)
	for i, portStr := range inPorts {
		port, err := strconv.ParseUint(portStr, 10, 64)
		if err != nil {
			return nil, err
		}
		if port >= 65536 {
			return nil, fmt.Errorf("port must be < 65536 but got %d", port)
		}
		outPorts[i] = Port{
			ContainerPort: int32(port),
			ExternalPort: int32(port),
			Protocol: "TCP",
		}
	}
	return outPorts, nil
} 
