package activity_network_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/activity_network"
)

func TestProbes_ReturnNothingForAnUnusableURLOrAnAnswerThatIsNotJSON(t *testing.T) {
	notJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "<html>not what a model server says</html>")
	}))
	defer notJSON.Close()

	for name, urls := range map[string][3]string{
		"a URL that cannot form a request": {"http://bad\x7fhost", "http://bad\x7fhost", "http://bad\x7fhost"},
		"answers that are not JSON":        {notJSON.URL, notJSON.URL, notJSON.URL},
	} {
		t.Run(name, func(t *testing.T) {
			r := activity_network.NewNetworkProbeReaderWithURLs(urls[0], urls[1], urls[2])
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			got, err := r.ReadActivities(ctx, time.Time{})
			if err != nil || len(got) != 0 {
				t.Errorf("activities=%+v err=%v; probing is best effort and reports nothing", got, err)
			}
		})
	}
}
