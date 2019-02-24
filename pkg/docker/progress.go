package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/docker/docker/pkg/jsonmessage"
)

type staticStatusInfo struct {
	label        string
	weight       float64
	weightBefore float64
}

const digestPrefix = "Digest: "

var (
	digestRegExp          = regexp.MustCompile("^" + regexp.QuoteMeta(digestPrefix) + "sha256:[a-f0-9A-F]{64}$")
	staticStatusInfoSlice = []*staticStatusInfo{
		&staticStatusInfo{
			label:  "Waiting",
			weight: 1,
		},
		&staticStatusInfo{
			label:  "Pulling fs layer",
			weight: 1,
		},
		&staticStatusInfo{
			label:  "Downloading",
			weight: 20,
		},
		&staticStatusInfo{
			label:  "Download complete",
			weight: 1,
		},
		&staticStatusInfo{
			label:  "Verifying checksum",
			weight: 1,
		},
		&staticStatusInfo{
			label:  "Extracting",
			weight: 5,
		},
		&staticStatusInfo{
			label:  "Pull complete",
			weight: 1,
		},
	}
	statusInfoFromLabel map[string]*staticStatusInfo
	maxWeight           float64
)

// Special init function for this package
func init() {
	n := len(staticStatusInfoSlice)
	statusInfoFromLabel = make(map[string]*staticStatusInfo, n)
	if n == 0 {
		return
	}
	for i := 1; i < n; i++ {
		staticStatusInfoSlice[i].weightBefore = staticStatusInfoSlice[i-1].weightBefore + staticStatusInfoSlice[i-1].weight
	}
	maxWeight = staticStatusInfoSlice[n-1].weightBefore + staticStatusInfoSlice[n-1].weight

	for _, item := range staticStatusInfoSlice {
		statusInfoFromLabel[item.label] = item
	}
}

type PullOrPush struct {
	reader          io.Reader
	statusFromLayer map[string]*status
}

type status struct {
	statusEnum *staticStatusInfo
	progress   *jsonmessage.JSONProgress
}

func NewPullOrPush(r io.Reader) *PullOrPush {
	return &PullOrPush{
		reader:          r,
		statusFromLayer: map[string]*status{},
	}
}

func (d *PullOrPush) GetProgress() float64 {
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
			layerProgress = weight / maxWeight
		}
		sum = sum + layerProgress
		count = count + 1
	}
	return sum / float64(count)
}

func (d *PullOrPush) Wait(onUpdate func(*PullOrPush)) (string, error) {
	decoder := json.NewDecoder(d.reader)
	digest := ""
	for {
		var msg jsonmessage.JSONMessage
		err := decoder.Decode(&msg)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		statusEnum := statusInfoFromLabel[msg.Status]
		fmt.Printf("%+v\n", msg)
		if statusEnum != nil {
			s := d.statusFromLayer[msg.ID]
			if s == nil {
				s = &status{}
				d.statusFromLayer[msg.ID] = s
			}
			s.statusEnum = statusEnum
			s.progress = msg.Progress
			onUpdate(d)
		} else if digestRegExp.MatchString(msg.Status) {
			digest = msg.Status[len(digestPrefix):]
		}
	}
	return digest, nil
}
