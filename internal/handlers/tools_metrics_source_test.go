package handlers

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matthieu/mcp-server-prtg/internal/database"
	"github.com/matthieu/mcp-server-prtg/internal/prtg"
	"github.com/matthieu/mcp-server-prtg/internal/types"
)

type fakeChannelStore struct {
	lastValues      []types.ChannelLastValue
	lastErr         error
	lastValuesCalls int
	lastMaxAge      time.Duration

	history      *types.ChannelHistory
	histErr      error
	historyCalls int
	lastFrom     time.Time
	lastTo       time.Time
	lastBucket   time.Duration
}

func (f *fakeChannelStore) GetChannelLastValues(ctx context.Context, sensorID int, maxAge time.Duration) ([]types.ChannelLastValue, error) {
	f.lastValuesCalls++
	f.lastMaxAge = maxAge
	return f.lastValues, f.lastErr
}

func (f *fakeChannelStore) GetChannelHistory(ctx context.Context, sensorID int, from, to time.Time, bucket time.Duration) (*types.ChannelHistory, error) {
	f.historyCalls++
	f.lastFrom = from
	f.lastTo = to
	f.lastBucket = bucket
	return f.history, f.histErr
}

type fakePRTGClient struct {
	tsData  *prtg.TimeSeriesData
	tsErr   error
	tsCalls int

	chData  []prtg.Channel
	chErr   error
	chCalls int
}

func (f *fakePRTGClient) GetTimeSeries(ctx context.Context, objectID int, timeType prtg.TimeSeriesType) (*prtg.TimeSeriesData, error) {
	f.tsCalls++
	return f.tsData, f.tsErr
}

func (f *fakePRTGClient) GetChannelsBySensor(ctx context.Context, sensorID int) ([]prtg.Channel, error) {
	f.chCalls++
	return f.chData, f.chErr
}

func (f *fakePRTGClient) GetSensors(ctx context.Context, filters map[string]interface{}) ([]prtg.Sensor, error) {
	return nil, nil
}
func (f *fakePRTGClient) GetSensorByID(ctx context.Context, sensorID int) (*prtg.Sensor, error) {
	return nil, nil
}
func (f *fakePRTGClient) GetDeviceByID(ctx context.Context, deviceID int, includes ...string) (*prtg.Device, error) {
	return nil, nil
}
func (f *fakePRTGClient) GetGroups(ctx context.Context, filters map[string]interface{}) ([]prtg.Group, error) {
	return nil, nil
}
func (f *fakePRTGClient) GetObjects(ctx context.Context, filters map[string]interface{}) ([]prtg.PRTGObject, error) {
	return nil, nil
}
func (f *fakePRTGClient) GetObjectCounts(ctx context.Context) (*prtg.SensorCount, error) {
	return nil, nil
}
func (f *fakePRTGClient) GetStatusSummary(ctx context.Context) (*prtg.StatusSummary, error) {
	return nil, nil
}

func newTestMetricsHandler(client PRTGClient, store ChannelStore) *MetricsToolHandler {
	l := zerolog.New(io.Discard)
	return NewMetricsToolHandlerWithStore(client, store, NewToolHandler(nil, nil, &l))
}

func callMetricsTool(t *testing.T, h *MetricsToolHandler, timeseries bool, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	var res *mcp.CallToolResult
	var err error
	if timeseries {
		res, err = h.handleGetSensorTimeSeries(context.Background(), req)
	} else {
		res, err = h.handleGetChannelCurrentValues(context.Background(), req)
	}
	require.NoError(t, err)
	require.NotEmpty(t, res.Content)
	return res
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	return res.Content[0].(mcp.TextContent).Text
}

func testHistory() *types.ChannelHistory {
	now := time.Now().UTC().Truncate(time.Minute)
	return &types.ChannelHistory{
		SensorID: 1001,
		From:     now.Add(-24 * time.Hour),
		To:       now,
		Bucket:   15 * time.Minute,
		Channels: []types.ChannelInfo{
			{ID: 1, Name: "Traffic In", Unit: "kbit/s"},
			{ID: 2, Name: "Traffic Out", Unit: "kbit/s"},
		},
		Points: []types.ChannelPoint{
			{Timestamp: now.Add(-15 * time.Minute), Values: map[int]float64{1: 10.5, 2: 20.25}},
			{Timestamp: now, Values: map[int]float64{1: 11, 2: 21}},
		},
	}
}

func testPRTGTimeSeries() *prtg.TimeSeriesData {
	now := time.Now()
	return &prtg.TimeSeriesData{
		ObjectID: 1001,
		TimeType: prtg.TimeSeriesMedium,
		Headers:  []string{"Timestamp", "Traffic"},
		DataPoints: []prtg.TimeSeriesDataPoint{
			{Timestamp: now, Values: map[string]interface{}{"Traffic": 1.5}},
		},
	}
}

func TestTimeSeriesAutoUsesStoreFirst(t *testing.T) {
	store := &fakeChannelStore{history: testHistory()}
	client := &fakePRTGClient{tsData: testPRTGTimeSeries()}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	assert.Equal(t, 0, client.tsCalls)
	assert.Equal(t, 1, store.historyCalls)
	assert.Contains(t, resultText(t, res), "Source: PRTG Data Exporter database")
}

func TestTimeSeriesAutoFallsBackToPRTGOnNoChannelData(t *testing.T) {
	store := &fakeChannelStore{histErr: database.ErrNoChannelData}
	client := &fakePRTGClient{tsData: testPRTGTimeSeries()}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "medium"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.tsCalls)
	assert.Contains(t, resultText(t, res), "Source: PRTG API v2 (real time)")
}

func TestTimeSeriesAutoFallsBackToPRTGOnSensorNotFound(t *testing.T) {
	store := &fakeChannelStore{histErr: database.ErrSensorNotFound}
	client := &fakePRTGClient{tsData: testPRTGTimeSeries()}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "long"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.tsCalls)
}

func TestTimeSeriesAutoFallsBackToPRTGOnEmptyHistory(t *testing.T) {
	store := &fakeChannelStore{history: &types.ChannelHistory{SensorID: 1001, Points: nil}}
	client := &fakePRTGClient{tsData: testPRTGTimeSeries()}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.tsCalls)
}

func TestTimeSeriesLivePrefersPRTG(t *testing.T) {
	store := &fakeChannelStore{history: testHistory()}
	client := &fakePRTGClient{tsData: testPRTGTimeSeries()}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "live"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.tsCalls)
	assert.Equal(t, 0, store.historyCalls)
	assert.Contains(t, resultText(t, res), "Source: PRTG API v2 (real time)")
}

func TestTimeSeriesLiveWithoutClientUsesStore(t *testing.T) {
	store := &fakeChannelStore{history: testHistory()}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "live"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, store.historyCalls)
	assert.Equal(t, 5*time.Minute, store.lastBucket)
	assert.InDelta(t, (2 * time.Hour).Seconds(), store.lastTo.Sub(store.lastFrom).Seconds(), 2)
	assert.Contains(t, resultText(t, res), "Source: PRTG Data Exporter database")
}

func TestTimeSeriesSourcePRTGWithoutClient(t *testing.T) {
	store := &fakeChannelStore{history: testHistory()}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short", "source": "prtg"})
	assert.True(t, res.IsError)
	assert.Equal(t, 0, store.historyCalls)
}

func TestTimeSeriesSourceDatabaseNeverCallsPRTG(t *testing.T) {
	store := &fakeChannelStore{history: &types.ChannelHistory{SensorID: 1001, Points: nil}}
	client := &fakePRTGClient{tsData: testPRTGTimeSeries()}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short", "source": "database"})
	assert.False(t, res.IsError)
	assert.Equal(t, 0, client.tsCalls)
	assert.Equal(t, 1, store.historyCalls)
}

func TestTimeSeriesLabelCollisionSuffix(t *testing.T) {
	hist := testHistory()
	hist.Channels = []types.ChannelInfo{
		{ID: 7, Name: "Traffic", Unit: "kbit"},
		{ID: 9, Name: "Traffic", Unit: "kbit"},
	}
	hist.Points = []types.ChannelPoint{
		{Timestamp: time.Now(), Values: map[int]float64{7: 1, 9: 2}},
	}
	store := &fakeChannelStore{history: hist}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	text := resultText(t, res)
	assert.Contains(t, text, "Traffic (kbit)")
	assert.Contains(t, text, "Traffic (kbit) #9")
}

func TestCurrentValuesAutoUsesStore(t *testing.T) {
	store := &fakeChannelStore{lastValues: []types.ChannelLastValue{
		{ChannelInfo: types.ChannelInfo{ID: 1, Name: "CPU", Unit: "%"}, Value: 55.5, Timestamp: time.Now()},
	}}
	client := &fakePRTGClient{}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.False(t, res.IsError)
	assert.Equal(t, 0, client.chCalls)
	text := resultText(t, res)
	assert.Contains(t, text, "Source: PRTG Data Exporter database")
	assert.Contains(t, text, "| CPU | 55.50 | % |")
	assert.NotContains(t, text, "Note: the newest exported value")
}

func TestCurrentValuesStaleNote(t *testing.T) {
	store := &fakeChannelStore{lastValues: []types.ChannelLastValue{
		{ChannelInfo: types.ChannelInfo{ID: 1, Name: "CPU", Unit: "%"}, Value: 55.5, Timestamp: time.Now().Add(-3 * time.Hour)},
	}}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.False(t, res.IsError)
	assert.Contains(t, resultText(t, res), "Note: the newest exported value is 3h0m0s old; use source=prtg for real-time values.")
}

func TestCurrentValuesAutoFallsBackToPRTG(t *testing.T) {
	store := &fakeChannelStore{lastErr: database.ErrNoChannelData}
	client := &fakePRTGClient{}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.chCalls)
	assert.Contains(t, resultText(t, res), "Source: PRTG API v2 (real time)")
}

func TestCurrentValuesEmptyWithoutClient(t *testing.T) {
	store := &fakeChannelStore{}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.False(t, res.IsError)
	assert.Contains(t, resultText(t, res), "No exported channel values for sensor 1001 in the last 7 days")
}

func TestCurrentValuesSourceDatabaseNeverCallsPRTG(t *testing.T) {
	store := &fakeChannelStore{}
	client := &fakePRTGClient{}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001, "source": "database"})
	assert.False(t, res.IsError)
	assert.Equal(t, 0, client.chCalls)
	assert.Equal(t, 1, store.lastValuesCalls)
	assert.Equal(t, 7*24*time.Hour, store.lastMaxAge)
}

func TestCurrentValuesStoreErrorIsToolError(t *testing.T) {
	store := &fakeChannelStore{lastErr: errors.New("connection lost")}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.True(t, res.IsError)
}
