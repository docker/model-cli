package metrics

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/docker/model-distribution/types"
	"github.com/docker/model-runner/pkg/logging"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
)

type Tracker struct {
	doNotTrack bool
	transport  http.RoundTripper
	log        logging.Logger
	userAgent  string
}

type TrackerRoundTripper struct {
	Transport http.RoundTripper
}

func (h *TrackerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clonedReq := req.Clone(req.Context())
	ctx, cancel := context.WithTimeout(clonedReq.Context(), 5*time.Second)
	defer cancel()
	clonedReq = clonedReq.WithContext(ctx)
	clonedReq.Header.Set("x-docker-model-runner", "true")
	return h.Transport.RoundTrip(clonedReq)
}

func NewTracker(httpClient *http.Client, log logging.Logger, userAgent string, doNotTrack bool) *Tracker {
	client := *httpClient
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}

	if userAgent == "" {
		userAgent = "docker-model-runner"
	}

	if os.Getenv("DEBUG") == "1" {
		if logger, ok := log.(*logrus.Logger); ok {
			logger.SetLevel(logrus.DebugLevel)
		} else if entry, ok := log.(*logrus.Entry); ok {
			entry.Logger.SetLevel(logrus.DebugLevel)
		}
	}

	return &Tracker{
		doNotTrack: os.Getenv("DO_NOT_TRACK") == "1" || doNotTrack,
		transport:  &TrackerRoundTripper{Transport: client.Transport},
		log:        log,
		userAgent:  userAgent,
	}
}

func (t *Tracker) TrackModel(model types.Model) {
	if t.doNotTrack {
		return
	}

	go t.trackModel(model)
}

func (t *Tracker) trackModel(model types.Model) {
	tags := model.Tags()
	t.log.Debugln("Tracking model:", tags)
	if len(tags) == 0 {
		return
	}
	ref, err := name.ParseReference(tags[0])
	if err != nil {
		t.log.Errorf("Error parsing reference: %v\n", err)
		return
	}
	if _, err = remote.Head(ref,
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
		remote.WithTransport(t.transport),
		remote.WithUserAgent(t.userAgent),
	); err != nil {
		t.log.Debugf("Manifest does not exist or error occurred: %v\n", err)
		return
	}
	t.log.Debugln("Tracked", ref.Name(), ref.Identifier())
}
