package config

import (
	"fmt"
	"regexp"
	"strconv"
<<<<<<< HEAD

	"github.com/pkg/errors"
)

// TODO https://github.com/jbrekelmans/kube-compose/issues/46
var portBindingSpecRegexp = regexp.MustCompile(
	"^" + // Match full string
		"(?:" + // External part
		"(?:(?P<host>[a-fA-F\\d.:]+?):)?" + // IP address
		"(?P<externalMin>[\\d]*)(?:-(?P<externalMax>\\d+))?:" + // External range
		")?" +
		"(?P<internalMin>\\d+)(?:-(?P<internalMax>\\d+))?" + // Internal range
		"(?P<protocol>/(?:udp|tcp|sctp))?" + // Protocol
		"$", // Match full string)
=======
	"strings"

	"github.com/pkg/errors"
>>>>>>> 27b1745... fix #15, #14 and #17
)

// PortBinding is the parsed/canonical form of a docker publish port specification.
type PortBinding struct {
	// the internal port; the port on which the container would listen. At least 0 and less than 65536.
	Internal int32
	// the minimum external port. At least -1 and less than 65536. -1 if the internal port is not published.
	ExternalMin int32
	// the maximum external port. This value is undefined if ExternalMin is -1. Otherwise, at least 0 and less than 65536.
	// Docker will choose from a random available port from the range to map to the internal port.
	ExternalMax int32
	// one of "udp", "tcp" and "sctp"
	Protocol string
	// the host (see docker for more details). Can be an empty string if the host was not set in the specification.
	Host string
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
<<<<<<< HEAD
// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func parsePortBindings(spec string, portBindings []PortBinding) ([]PortBinding, error) {
	matches := portBindingSpecRegexp.FindStringSubmatch(spec)
	if matches == nil {
		return nil, fmt.Errorf("invalid port %q, should be [[remote_ip:]remote_port[-remote_port]:]port[/protocol]", spec)
	}
	matchMap := buildRegexpMatchMap(portBindingSpecRegexp, matches)

	host := matchMap["host"]
	protocol := matchMap["protocol"]
	if protocol == "" {
		protocol = "tcp"
	}

	internal := []int32{}
	internalMin, err := parsePortUint(matchMap["internalMin"])
	if err != nil {
		return nil, err
	}
	internalMaxStr := matchMap["internalMax"]
	if internalMaxStr == "" {
		internal = append(internal, internalMin)
	} else {
		internalMax, err := parsePortUint(internalMaxStr)
		if err != nil {
			return nil, err
		}
		for i := internalMin; i <= internalMax; i++ {
			internal = append(internal, i)
		}
	}

	external := []int32{}
	externalMinStr := matchMap["externalMin"]
	if len(externalMinStr) > 0 {
		externalMin, err := parsePortUint(externalMinStr)
		if err != nil {
			return nil, err
		}
		externalMaxStr := matchMap["externalMax"]
		if externalMaxStr == "" {
			external = append(external, externalMin)
		} else {
			externalMax, err := parsePortUint(externalMaxStr)
			if err != nil {
				return nil, err
			}
			if len(internal) == 1 {
				return append(portBindings, PortBinding{
					Internal:    internal[0],
					ExternalMin: externalMin,
					ExternalMax: externalMax,
					Protocol:    protocol,
					Host:        host,
				}), nil
			}
			for i := externalMin; i <= externalMax; i++ {
				external = append(external, i)
			}
		}
	}
	if len(externalMinStr) > 0 && len(internal) != len(external) {
		return nil, fmt.Errorf("port ranges don't match in length")
	}
	for j, i := range internal {
		portBinding := PortBinding{
			Internal:    i,
			ExternalMin: -1,
			ExternalMax: -1,
			Protocol:    protocol,
			Host:        host,
		}
		if len(externalMinStr) > 0 {
			portBinding.ExternalMin = external[j]
			portBinding.ExternalMax = external[j]
		}
		portBindings = append(portBindings, portBinding)
	}
	return portBindings, nil
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

func parsePorts(inputs []port) ([]PortBinding, error) {
	portBindings := []PortBinding{}
	for _, input := range inputs {
		var err error
		portBindings, err = parsePortBindings(input.Value, portBindings)
		if err != nil {
			return nil, err
=======
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
>>>>>>> 27b1745... fix #15, #14 and #17
		}
	}
	return portBindings, nil
}
