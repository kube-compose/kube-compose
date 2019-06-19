package config

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/kube-compose/kube-compose/internal/pkg/util"
	"github.com/pkg/errors"
)

// TODO https://github.com/kube-compose/kube-compose/issues/46
var portBindingSpecRegexp = regexp.MustCompile(
	"^" + // Match full string
		"(?:" + // External part
		"(?:(?P<host>[a-fA-F\\d.:]+?):)?" + // IP address
		"(?P<externalMin>[\\d]*)(?:-(?P<externalMax>\\d+))?:" + // External range
		")?" +
		"(?P<internalMin>\\d+)(?:-(?P<internalMax>\\d+))?" + // Internal range
		"(?P<protocol>/(?:udp|tcp|sctp))?" + // Protocol
		"$", // Match full string)
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

type portBindingParser struct {
	external       []int32
	externalMinStr string
	internal       []int32
	host           string
	protocol       string
	done           bool
	result         []PortBinding
}

func (parser *portBindingParser) parseInternal(matchMap map[string]string) error {
	internalMin, err := parsePortUint(matchMap["internalMin"])
	if err != nil {
		return err
	}
	internalMaxStr := matchMap["internalMax"]
	if internalMaxStr == "" {
		parser.internal = append(parser.internal, internalMin)
	} else {
		internalMax, err := parsePortUint(internalMaxStr)
		if err != nil {
			return err
		}
		for i := internalMin; i <= internalMax; i++ {
			parser.internal = append(parser.internal, i)
		}
	}
	return nil
}

func (parser *portBindingParser) parseExternal(matchMap map[string]string) error {
	parser.externalMinStr = matchMap["externalMin"]
	if len(parser.externalMinStr) > 0 {
		externalMin, err := parsePortUint(parser.externalMinStr)
		if err != nil {
			return err
		}
		externalMaxStr := matchMap["externalMax"]
		if externalMaxStr == "" {
			parser.external = append(parser.external, externalMin)
		} else {
			externalMax, err := parsePortUint(externalMaxStr)
			if err != nil {
				return err
			}
			if len(parser.internal) == 1 {
				parser.result = append(parser.result, PortBinding{
					Internal:    parser.internal[0],
					ExternalMin: externalMin,
					ExternalMax: externalMax,
					Protocol:    parser.protocol,
					Host:        parser.host,
				})
				parser.done = true
				return nil
			}
			for i := externalMin; i <= externalMax; i++ {
				parser.external = append(parser.external, i)
			}
		}
	}
	return nil
}

func (parser *portBindingParser) buildResult() error {
	if len(parser.externalMinStr) > 0 && len(parser.internal) != len(parser.external) {
		return fmt.Errorf("port ranges don't match in length")
	}
	for j, i := range parser.internal {
		portBinding := PortBinding{
			Internal:    i,
			ExternalMin: -1,
			ExternalMax: -1,
			Protocol:    parser.protocol,
			Host:        parser.host,
		}
		if len(parser.externalMinStr) > 0 {
			portBinding.ExternalMin = parser.external[j]
			portBinding.ExternalMax = parser.external[j]
		}
		parser.result = append(parser.result, portBinding)
	}
	return nil
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
func parsePortBindings(spec string, portBindings []PortBinding) ([]PortBinding, error) {
	parser := portBindingParser{
		result: portBindings,
	}
	matches := portBindingSpecRegexp.FindStringSubmatch(spec)
	if matches == nil {
		return nil, fmt.Errorf("invalid port %q, should be [[remote_ip:]remote_port[-remote_port]:]port[/protocol]", spec)
	}
	matchMap := util.BuildRegexpMatchMap(portBindingSpecRegexp, matches)

	parser.host = matchMap["host"]
	parser.protocol = matchMap["protocol"]
	if parser.protocol == "" {
		parser.protocol = "tcp"
	} else {
		parser.protocol = parser.protocol[1:]
	}

	err := parser.parseInternal(matchMap)
	if err != nil {
		return nil, err
	}
	err = parser.parseExternal(matchMap)
	if err != nil {
		return nil, err
	}
	if parser.done {
		return parser.result, nil
	}
	err = parser.buildResult()
	return parser.result, err
}

func parsePortUint(portStr string) (int32, error) {
	port, err := strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		return -1, errors.Wrapf(err, "unsupported port format %s", portStr)
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
		}
	}
	return portBindings, nil
}
