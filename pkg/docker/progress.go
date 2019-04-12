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

const sha256Prefix = "sha256:"
const sha256BitLength = 256

var (
	digestRegExp = regexp.MustCompile(
		fmt.Sprintf(
			"%s[a-fA-F0-9]{%d}(?:[^a-fA-F0-9]|$)",
			regexp.QuoteMeta(sha256Prefix),
			sha256BitLength/4,
		),
	)
	maxPullWeight             float64
	maxPushWeight             float64
	staticPullStatusInfoSlice = []*staticStatusInfo{
		&staticStatusInfo{
			labels: []string{"Waiting"},
			weight: 1,
		},
		&staticStatusInfo{
			labels: []string{"Pulling fs layer"},
			weight: 1,
		},
		&staticStatusInfo{
			labels: []string{"Downloading"},
			weight: 20,
		},
		&staticStatusInfo{
			labels: []string{"Verifying checksum"},
			weight: 1,
		},
		&staticStatusInfo{
			labels: []string{"Download complete"},
			weight: 1,
		},
		&staticStatusInfo{
			labels: []string{"Extracting"},
			weight: 5,
		},
		&staticStatusInfo{
			labels: []string{"Pull complete", "Already exists"},
			weight: 1,
		},
	}
	staticPushStatusInfoSlice = []*staticStatusInfo{
		&staticStatusInfo{
			labels: []string{"Waiting"},
			weight: 1,
		},
		&staticStatusInfo{
			labels: []string{"Preparing"},
			weight: 1,
		},
		&staticStatusInfo{
			labels: []string{"Pushing"},
			weight: 20,
		},
		&staticStatusInfo{
			labels: []string{"Layer already exists", "Pushed"},
			weight: 1,
		},
	}
	staticPullStatusInfoFromLabel map[string]*staticStatusInfo
	staticPushStatusInfoFromLabel map[string]*staticStatusInfo
)

// Special init function for this package
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
		layerProgress := 0.0
		if status.statusEnum != nil {
			weight := status.statusEnum.weightBefore
			if status.progress != nil && status.progress.Total > 0 {
				statusProgress := float64(status.progress.Current) / float64(status.progress.Total)
				weight = weight + (statusProgress * status.statusEnum.weight)
			}
			layerProgress = weight / d.maxWeight
		}
		sum = sum + layerProgress
		count = count + 1
	}
	return sum / float64(count)
}

func (d *PullOrPush) Wait(onUpdate func(*PullOrPush)) (string, error) {
	decoder := json.NewDecoder(d.reader)
	digest := ""
	var lastError string
	for {
		var msg jsonmessage.JSONMessage
		err := decoder.Decode(&msg)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		statusEnum := d.staticStatusInfoFromLabel[msg.Status]
		if statusEnum != nil {
			s := d.statusFromLayer[msg.ID]
			if s == nil {
				s = &status{}
				d.statusFromLayer[msg.ID] = s
			}
			s.statusEnum = statusEnum
			s.progress = msg.Progress
			onUpdate(d)
			// TODO https://github.com/jbrekelmans/kube-compose/issues/5 support non-sha256 digests
		} else if loc := digestRegExp.FindStringIndex(msg.Status); loc != nil {
			len := sha256BitLength/4 + len(sha256Prefix)
			digest = msg.Status[loc[0] : loc[0]+len]
		} else if msg.Error != nil && len(msg.Error.Message) > 0 {
			lastError = msg.Error.Message
		}
	}
	if len(digest) == 0 {
		verb := "pushing"
		if d.isPull {
			verb = "pulling"
		}
		if len(lastError) > 0 {
			return "", fmt.Errorf("error while %s image: %s", verb, lastError)
		}
		return "", fmt.Errorf("unknown error while %s image", verb)
	}
	return digest, nil
}
