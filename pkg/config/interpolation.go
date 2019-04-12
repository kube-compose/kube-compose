package config

import (
	"fmt"
	"reflect"
	"strings"

<<<<<<< HEAD
	version "github.com/hashicorp/go-version"
=======
	"github.com/hashicorp/go-version"
>>>>>>> d2d10a0... finalize implementation of variable substitutions
)

type ValueGetter func(name string) (string, bool)

type configInterpolator struct {
	config      genericMap
	errorList   []error
	fileName    string
	valueGetter ValueGetter
	version     *version.Version
<<<<<<< HEAD
}

type stringOrInt struct {
	str   string
	i     int
	isInt bool
}

type path []stringOrInt

func (p path) appendStr(str string) path {
	return append(p, stringOrInt{
		str: str,
	})
}

func (p path) appendInt(i int) path {
	return append(p, stringOrInt{
		i:     i,
		isInt: true,
	})
}

func (p path) pop() path {
	return p[:len(p)-1]
}

func (c *configInterpolator) run() error {
	if !c.version.GreaterThan(v1) {
		c.interpolateSection(c.config, path{})
	} else {
		c.interpolateSectionByName("services")
		if !c.version.LessThan(v3_1) {
			c.interpolateSectionByName("volumes")
			c.interpolateSectionByName("networks")
			if !c.version.LessThan(v3_3) {
				c.interpolateSectionByName("secrets")
				c.interpolateSectionByName("configs")
			}
		}
	}
=======
}

type stringOrInt struct {
	str string
	i   int
}

type path []stringOrInt

func (p path) appendStr(str string) path {
	if len(str) == 0 {
		panic(fmt.Errorf("s must not be empty"))
	}
	return append(p, stringOrInt{
		str: str,
	})
}

func (p path) appendInt(i int) path {
	return append(p, stringOrInt{
		i: i,
	})
}

func (p path) pop() path {
	return p[:len(p)-1]
}

func (c *configInterpolator) run() error {
	if !c.version.GreaterThan(v1) {
		c.interpolateSection(c.config, path{})
	} else {
		c.interpolateSectionByName("services")
		if !c.version.LessThan(v3_1) {
			c.interpolateSectionByName("volumes")
			c.interpolateSectionByName("networks")
			if !c.version.LessThan(v3_3) {
				c.interpolateSectionByName("secrets")
				c.interpolateSectionByName("configs")
			}
		}
	}
>>>>>>> d2d10a0... finalize implementation of variable substitutions
	// Primitive error handling does not report all errors...
	if len(c.errorList) > 0 {
		return c.errorList[0]
	}
	return nil
}

func (c *configInterpolator) interpolateSectionByName(name string) {
	if sectionRaw, ok := c.config[name]; ok {
		if section, ok := sectionRaw.(genericMap); ok {
			c.interpolateSection(section, (path{}).appendStr(name))
		}
	}
}

func (c *configInterpolator) interpolateSection(configDict genericMap, p path) {
	for keyRaw, val := range configDict {
		if key, ok := keyRaw.(string); ok {
			childPath := p.appendStr(key)
			val2 := c.interpolateRecursive(val, childPath)
			configDict[key] = val2
			childPath.pop()
		}
	}
}

<<<<<<< HEAD
func (c *configInterpolator) addError(err error, _ path) {
=======
func (c *configInterpolator) addError(err error, p path) {
>>>>>>> d2d10a0... finalize implementation of variable substitutions
	c.errorList = append(c.errorList, err)
}

// InterpolateConfig takes the root of a docker compose file as a generic structure and substitutes variables in it.
<<<<<<< HEAD
// The implementation substitutes exactly the same sections as docker compose:
// https://github.com/docker/compose/master/compose/config/config.py.
// TODO https://github.com/jbrekelmans/kube-compose/issues/11 support arbitrary map types instead of genericMap.
func InterpolateConfig(fileName string, config genericMap, valueGetter ValueGetter, v *version.Version) error {
=======
// The implementation substitutes exactly the same sections as docker compose: https://github.com/docker/compose/blob/master/compose/config/config.py.
// TODO https://github.com/jbrekelmans/kube-compose/issues/11 support arbitrary map types instead of genericMap.
func InterpolateConfig(fileName string, config genericMap, valueGetter ValueGetter, version *version.Version) error {
>>>>>>> d2d10a0... finalize implementation of variable substitutions
	c := &configInterpolator{
		config:      config,
		fileName:    fileName,
		valueGetter: valueGetter,
		version:     v,
	}
	return c.run()
}

// Interpolate substitutes docker-compose style variables in the str.
// The docker-compose 2.1+ syntax is used if and only if version is true.
// The implementation is not strict on the syntax between two paired curly braces, but
// is otherwise identical to the Python implementation:
// https://github.com/docker/compose/blob/master/compose/config/interpolation.py
// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func Interpolate(str string, valueGetter ValueGetter, v bool) (string, error) {
	var sb strings.Builder
	for {
		i := strings.IndexRune(str, '$')
		if i < 0 {
			break
		}
		sb.WriteString(str[:i])
		str = str[i+1:]
		if str == "" {
			return "", fmt.Errorf("$ followed by EOF")
		}
		if str[0] == byte('_') || IsASCIILetter(str[0]) {
			// The grammar of names without braces is [a-zA-Z_][a-zA-Z0-9_]+
			// Scan until no letter, digit or underscore to find end of placeholder...
			i = 1
			for i < len(str) && (str[i] == byte('_') || IsASCIILetter(str[i]) || IsASCIIDigit(str[i])) {
				i++
			}
			value, found := valueGetter(str[0:i])
			if !found {
				value = ""
			}
			sb.WriteString(value)
			str = str[i:]
			continue
		}
		if str[0] == byte('{') {
			// Scan until '}' to perform substitution...
			i = strings.IndexRune(str[1:], '}')
			if i < 0 {
				return "", fmt.Errorf("expected }")
			}
			i++

			// Process what is between the two curly braces
			j := -1
			treatEmptyAsUnset := false
			hasErrorMsg := false
			var errorMsgOrDefaultVal string
			if v {
				j = strings.IndexAny(str[1:i], ":?-")
				if j >= 0 {
					j++
					switch {
					case str[j] == byte(':'):
						treatEmptyAsUnset = true
						switch {
						case str[j+1] == byte('?'):
							hasErrorMsg = true
							errorMsgOrDefaultVal = str[j+2 : i]
						case str[j+1] == byte('-'):
							errorMsgOrDefaultVal = str[j+2 : i]
						default:
							j = -1
						}
					case str[j] == byte('?'):
						hasErrorMsg = true
						errorMsgOrDefaultVal = str[j+1 : i]
					default:
						errorMsgOrDefaultVal = str[j+1 : i]
					}
				}
			}
			if j < 0 {
				value, found := valueGetter(str[1:i])
				if !found {
					value = ""
				}
				sb.WriteString(value)
				str = str[i+1:]
				continue
			}
			name := str[1:j]
			value, found := valueGetter(name)
			if !found || (value == "" && treatEmptyAsUnset) {
				if hasErrorMsg {
					return "", fmt.Errorf("substitution variable %#v has no value or value is empty: %#v", name, errorMsgOrDefaultVal)
				}
				value = errorMsgOrDefaultVal
			}
			sb.WriteString(value)
			str = str[i+1:]
			continue
		}
		if str[0] == byte('$') {
			sb.WriteByte('$')
			str = str[1:]
			continue
		}
		return "", fmt.Errorf("unexpected character after $")
	}
	if sb.Len() == 0 {
		// Fast path
		return str, nil
	}
	// WriteString always returns a nil error
	sb.WriteString(str)
	return sb.String(), nil
}

// IsASCIILetter returns true if and only if b is the ASCII code for a letter.
func IsASCIILetter(b byte) bool {
	return (byte('a') <= b && b <= byte('z')) || (byte('A') <= b && b <= byte('Z'))
}

// IsASCIIDigit returns true if and only if b is the ASCII code for a digit.
func IsASCIIDigit(b byte) bool {
	return byte('0') <= b && b <= byte('9')
}

func (c *configInterpolator) interpolateRecursive(obj interface{}, p path) interface{} {
	if str, ok := obj.(string); ok {
		str2, err := Interpolate(str, c.valueGetter, !c.version.LessThan(v2_1))
		if err != nil {
			c.addError(err, p)
		}
		return str2
	}
	if m, ok := obj.(genericMap); ok {
		for keyRaw, val := range m {
			if key, ok := keyRaw.(string); ok {
				childPath := p.appendStr(key)
				m[key] = c.interpolateRecursive(val, childPath)
				childPath.pop()
			}
		}
		return m
	}
<<<<<<< HEAD
<<<<<<< HEAD
	if obj != nil && reflect.TypeOf(obj).Kind() == reflect.Slice {
		slicev := reflect.ValueOf(obj)
		for i := 0; i < slicev.Len(); i++ {
			iv := slicev.Index(i)
			childPath := p.appendInt(i)
			val := iv.Interface()
			val2 := c.interpolateRecursive(val, childPath)
			iv.Set(reflect.ValueOf(val2))
			childPath.pop()
		}
=======
	if slice, ok := obj.([]interface{}); ok {
		for i, val := range slice {
=======
	if reflect.TypeOf(obj).Kind() == reflect.Slice {
		slicev := reflect.ValueOf(obj)
		for i := 0; i < slicev.Len(); i++ {
			iv := slicev.Index(i)
>>>>>>> 32bf048... issue #7: fix bug where slices were not properly supported
			childPath := p.appendInt(i)
			val := iv.Interface()
			val2 := c.interpolateRecursive(val, childPath)
			iv.Set(reflect.ValueOf(val2))
			childPath.pop()
		}
<<<<<<< HEAD
		return slice
>>>>>>> d2d10a0... finalize implementation of variable substitutions
=======
>>>>>>> 32bf048... issue #7: fix bug where slices were not properly supported
	}
	return obj
}
