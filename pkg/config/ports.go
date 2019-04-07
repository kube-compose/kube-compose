package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Port struct {
	ContainerPort int32
	ExternalPort  int32
	Protocol      string
}

func parsePortUint(portStr string) (int32, error) {
	port, err := strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		return -1, errors.Wrap(err, fmt.Sprintf("unsupported port format %s", portStr))
	}
	if port >= 65536 {
		return -1, fmt.Errorf("port must be < 65536 but got %d", port)
	}
	return int32(port), nil
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
func parsePorts(inPorts []port) ([]Port, error) {
	n := len(inPorts)
	if n == 0 {
		return nil, nil
	}
	outPorts := make([]Port, n)
	for i, portRaw := range inPorts {
		portRawStr := portRaw.Value
		colonPos := strings.IndexByte(portRawStr, ':')
		var containerPort int32
		var externalPort int32
		if colonPos >= 0 {
			externalPortStr := portRawStr[:colonPos]
			containerPortStr := portRawStr[colonPos+1:]
			if strings.IndexByte(containerPortStr, ':') >= 0 {
				return nil, fmt.Errorf("unsupported port format %s", portRawStr)
			}
			var err error
			externalPort, err = parsePortUint(externalPortStr)
			if err != nil {
				return nil, err
			}
			containerPort, err = parsePortUint(containerPortStr)
			if err != nil {
				return nil, err
			}
		} else {
			var err error
			containerPort, err = parsePortUint(portRawStr)
			if err != nil {
				return nil, err
			}
			externalPort = containerPort
		}
		outPorts[i] = Port{
			ContainerPort: containerPort,
			ExternalPort:  externalPort,
			Protocol:      "TCP",
		}
	}
	return outPorts, nil
}
