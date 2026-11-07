package httpapi_test

import (
	"context"
	"errors"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// brokenWriter is a ResponseWriter whose connection is gone: every write fails. It has no Flush.
type brokenWriter struct{ header http.Header }

func (b *brokenWriter) Header() http.Header {
	if b.header == nil {
		b.header = http.Header{}
	}
	return b.header
}
func (b *brokenWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }
func (b *brokenWriter) WriteHeader(int)           {}

func TestConfigEndpoints_AnswerUnavailableWithoutAnOrchestrator(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(nil, nil)
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		if rec := send(h, method, "/api/config", `{"queueCap":5}`); rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s /api/config = %d, want 503", method, rec.Code)
		}
	}
}

func TestProviderEndpoints_RejectABodyThatIsNotJSON(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(&mockOrchestrator{}, nil)
	for _, path := range []string{"/api/providers", "/api/providers/config"} {
		if rec := send(h, http.MethodPost, path, "{broken"); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid JSON") {
			t.Errorf("POST %s = %d %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestWriteHelpers_LogAndSurviveAResponseThatCannotBeWritten(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	// A value JSON cannot encode (NaN) after the status line is out, and a client that went away.
	nan := &mockOrchestrator{getProvidersResult: []ports.ProviderInfo{{Name: "p", Quota: &domain.QuotaReading{Utilization: math.NaN()}}}}
	rec := send(realHandler(nan, nil), http.MethodGet, "/api/providers", "")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	realHandler(&mockOrchestrator{getTaskErr: domain.ErrNotFound}, nil).
		ServeHTTP(&brokenWriter{}, httptest.NewRequest(http.MethodGet, "/api/tasks/nope", nil))
}

func TestDashboard_SurvivesAClientThatCannotReceiveIt(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	realHandler(&mockOrchestrator{}, nil).ServeHTTP(&brokenWriter{}, httptest.NewRequest(http.MethodGet, "/ui", nil))
}

func TestEvents_RefuseAWriterThatCannotStream(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	bw := &brokenWriter{}
	realHandler(&mockOrchestrator{}, nil).ServeHTTP(bw, httptest.NewRequest(http.MethodGet, "/api/events", nil))
	if bw.Header().Get("Content-Type") != "application/json" {
		t.Errorf("a writer without Flush must get the JSON error, headers = %v", bw.Header())
	}
}

func TestHub_DropsAnEventItCannotEncode(t *testing.T) {
	hub := httpapi.NewHub()
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)
	hub.BroadcastAISessionEvent(domain.AISessionEvent{Type: "x", Timestamp: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)})
	select {
	case msg := <-ch:
		t.Errorf("an event with a year JSON cannot write was sent: %s", msg)
	default:
	}
	hub.BroadcastAISessionEvent(domain.AISessionEvent{Type: "x", Timestamp: time.Now()})
	select {
	case <-ch:
	default:
		t.Error("a valid event must be delivered")
	}
}

// brokerOrch is an orchestrator that takes the SSE hub, like the real service.
type brokerOrch struct {
	*mockOrchestrator
	got ports.EventBroadcaster
}

func (b *brokerOrch) SetBroadcaster(e ports.EventBroadcaster) { b.got = e }

func TestStartServer_GivesTheOrchestratorTheEventHubAndStopsWithItsContext(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	addr := freeAddr(t)
	orch := &brokerOrch{mockOrchestrator: &mockOrchestrator{}}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- httpapi.StartServer(ctx, orch, nil, addr) }()
	waitHTTP(t, "http://"+addr+"/api/health")
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("StartServer = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("StartServer did not stop")
	}
	if orch.got == nil {
		t.Error("the orchestrator never received the broadcaster")
	}
}

func TestRunHTTP_ClosesConnectionsThatOutliveTheGracePeriod(t *testing.T) {
	addr := freeAddr(t)
	release := make(chan struct{})
	defer close(release)
	started := make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			return
		}
		close(started)
		<-release // a request that ignores shutdown
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- httpapi.RunHTTP(ctx, addr, h, 50*time.Millisecond) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if resp, err := http.Get("http://" + addr + "/health"); err == nil {
			_ = resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server never started")
		}
		time.Sleep(20 * time.Millisecond)
	}
	go func() {
		if resp, err := http.Get("http://" + addr + "/slow"); err == nil {
			_ = resp.Body.Close()
		}
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("RunHTTP = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the server did not close the connection that outlived the grace period")
	}
}

func TestRunHTTP_ReportsAnAddressThatCannotBeServed(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	err = httpapi.RunHTTP(context.Background(), l.Addr().String(), http.NotFoundHandler(), time.Second)
	if err == nil {
		t.Fatal("an address that is already in use must be reported")
	}
}

// activityStub answers the activity endpoints and takes the SSE hub like the real service.
type activityStub struct{ got ports.ActivityBroadcaster }

func (a *activityStub) GetRecentActivities(context.Context, domain.ActivityFilter) ([]domain.AIActivity, error) {
	return nil, nil
}
func (a *activityStub) GetTimeline(context.Context, time.Time, int) ([]domain.AIActivity, error) {
	return nil, nil
}
func (a *activityStub) SetBroadcaster(b ports.ActivityBroadcaster) { a.got = b }

func TestStartServerWith_WiresTheActivityServiceTheLogHubAndTheMCPAddress(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	addr := freeAddr(t)
	act := &activityStub{}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- httpapi.StartServerWith(ctx, &brokerOrch{mockOrchestrator: &mockOrchestrator{}}, nil, addr, httpapi.Extras{
			Activity: act, LogHub: httpapi.NewLogHub(), MCPAddr: "127.0.0.1:9",
		})
	}()
	waitHTTP(t, "http://"+addr+"/api/health")
	resp, err := http.Get("http://" + addr + "/api/activities")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/api/activities = %d, want 200 with the service wired", resp.StatusCode)
	}
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("StartServerWith = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("StartServerWith did not stop")
	}
	if act.got == nil {
		t.Error("the activity service never received the broadcaster")
	}
}
