package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/docker/docker/pkg/jsonmessage"
)

type staticStatusInfo struct {
	labels       []string
	weight       float64
	weightBefore float64
}

const (
	sha256Prefix    = "sha256:"
	sha256BitLength = 256
)

var (
	digestRegexp = regexp.MustCompile(
		fmt.Sprintf(
			"%s[a-fA-F0-9]{%d}(?:[^a-fA-F0-9]|$)",
			regexp.QuoteMeta(sha256Prefix),
			sha256BitLength/4,
		),
	)
	maxPullWeight             float64
	maxPushWeight             float64
	staticPullStatusInfoSlice = []*staticStatusInfo{
		{
			labels: []string{"Waiting"},
			weight: 1,
		},
		{
			labels: []string{"Pulling fs layer"},
			weight: 1,
		},
		{
			labels: []string{"Downloading"},
			weight: 20,
		},
		{
			labels: []string{"Verifying checksum"},
			weight: 1,
		},
		{
			labels: []string{"Download complete"},
			weight: 1,
		},
		{
			labels: []string{"Extracting"},
			weight: 5,
		},
		{
			labels: []string{"Pull complete", "Already exists"},
			weight: 1,
		},
	}
	staticPushStatusInfoSlice = []*staticStatusInfo{
		{
			labels: []string{"Waiting"},
			weight: 1,
		},
		{
			labels: []string{"Preparing"},
			weight: 1,
		},
		{
			labels: []string{"Pushing"},
			weight: 20,
		},
		{
			labels: []string{"Layer already exists", "Pushed"},
			weight: 1,
		},
	}
	staticPullStatusInfoFromLabel map[string]*staticStatusInfo
	staticPushStatusInfoFromLabel map[string]*staticStatusInfo
)

// This init function is fine, ignoring linting warning.
// nolint
func init() {
	n := len(staticPullStatusInfoSlice)
	staticPullStatusInfoFromLabel = make(map[string]*staticStatusInfo, n)
	if n > 0 {
		for i := 1; i < n; i++ {
			staticPullStatusInfoSlice[i].weightBefore = staticPullStatusInfoSlice[i-1].weightBefore + staticPullStatusInfoSlice[i-1].weight
		}
		maxPullWeight = staticPullStatusInfoSlice[n-1].weightBefore + staticPullStatusInfoSlice[n-1].weight
		for _, item := range staticPullStatusInfoSlice {
			for _, label := range item.labels {
				staticPullStatusInfoFromLabel[label] = item
			}
		}
	}
	n = len(staticPushStatusInfoSlice)
	staticPushStatusInfoFromLabel = make(map[string]*staticStatusInfo, n)
	if n > 0 {
		for i := 1; i < n; i++ {
			staticPushStatusInfoSlice[i].weightBefore = staticPushStatusInfoSlice[i-1].weightBefore + staticPushStatusInfoSlice[i-1].weight
		}
		maxPushWeight = staticPushStatusInfoSlice[n-1].weightBefore + staticPushStatusInfoSlice[n-1].weight
		for _, item := range staticPushStatusInfoSlice {
			for _, label := range item.labels {
				staticPushStatusInfoFromLabel[label] = item
			}
		}
	}
}

type PullOrPush struct {
	isPull                    bool
	maxWeight                 float64
	reader                    io.Reader
	staticStatusInfoFromLabel map[string]*staticStatusInfo
	statusFromLayer           map[string]*status
}

type status struct {
	statusEnum *staticStatusInfo
	progress   *jsonmessage.JSONProgress
}

func NewPull(r io.Reader) *PullOrPush {
	return &PullOrPush{
		isPull:                    true,
		maxWeight:                 maxPullWeight,
		staticStatusInfoFromLabel: staticPullStatusInfoFromLabel,
		statusFromLayer:           map[string]*status{},
		reader:                    r,
	}
}

func NewPush(r io.Reader) *PullOrPush {
	return &PullOrPush{
		maxWeight:                 maxPushWeight,
		staticStatusInfoFromLabel: staticPushStatusInfoFromLabel,
		statusFromLayer:           map[string]*status{},
		reader:                    r,
	}
}

func (d *PullOrPush) Progress() float64 {
	if len(d.statusFromLayer) == 0 {
		return 0
	}
	sum := 0.0
	count := 0
	for _, status := range d.statusFromLayer {
		weight := status.statusEnum.weightBefore
		if status.progress != nil && status.progress.Total > 0 {
			statusProgress := float64(status.progress.Current) / float64(status.progress.Total)
			weight += (statusProgress * status.statusEnum.weight)
		} else {
			weight += status.statusEnum.weight
		}
		sum += weight / d.maxWeight
		count++
	}
	return sum / float64(count)
}

type pullOrPushWaiter struct {
	digest    string
	lastError string
	onUpdate  func(*PullOrPush)
}

func (waiter *pullOrPushWaiter) handleMessage(d *PullOrPush, msg *jsonmessage.JSONMessage) {
	statusEnum := d.staticStatusInfoFromLabel[msg.Status]
	if statusEnum != nil {
		s := d.statusFromLayer[msg.ID]
		if s == nil {
			s = &status{}
			d.statusFromLayer[msg.ID] = s
		}
		s.statusEnum = statusEnum
		s.progress = msg.Progress
		waiter.onUpdate(d)
	} else if digest := FindDigest(msg.Status); digest != "" {
		waiter.digest = digest
	} else if msg.Error != nil && len(msg.Error.Message) > 0 {
		waiter.lastError = msg.Error.Message
	}
}

// FindDigest finds a digest within a string. If it is found the digest is returned, otherwise returns the empty string.
func FindDigest(s string) string {
	// TODO https://github.com/kube-compose/kube-compose/issues/5 support non-sha256 digests
	loc := digestRegexp.FindStringIndex(s)
	if loc == nil {
		return ""
	}
	i := sha256BitLength/4 + len(sha256Prefix)
	return s[loc[0] : loc[0]+i]
}

func (waiter *pullOrPushWaiter) end(d *PullOrPush) (string, error) {
	if waiter.digest == "" {
		verb := "pushing"
		if d.isPull {
			verb = "pulling"
		}
		if waiter.lastError != "" {
			return "", fmt.Errorf("error while %s image: %s", verb, waiter.lastError)
		}
		return "", fmt.Errorf("unknown error while %s image", verb)
	}
	return waiter.digest, nil
}

// Wait processes a JSON stream (the body of an image pull docker HTTP response) and returns an error as soon as an error is encountered in
// the stream, or the digest could not be parsd aftere processing the entire stream. Otherwise, it returns the digest string and a no error.
// onUpdate is called whenever d.Progress() may return a different value from the previous call.
func (d *PullOrPush) Wait(onUpdate func(*PullOrPush)) (string, error) {
	waiter := pullOrPushWaiter{
		onUpdate: onUpdate,
	}
	decoder := json.NewDecoder(d.reader)
	for {
		var msg jsonmessage.JSONMessage
		err := decoder.Decode(&msg)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		waiter.handleMessage(d, &msg)
	}
	return waiter.end(d)
}
