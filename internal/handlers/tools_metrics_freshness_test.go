package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/senhub-io/mcp-server-prtg/internal/database"
	"github.com/senhub-io/mcp-server-prtg/internal/prtg"
	"github.com/senhub-io/mcp-server-prtg/internal/types"
)

func staleState() *types.SensorExportState {
	last := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	return &types.SensorExportState{SensorID: 1001, Tracked: true, LastFetchUTC: &last}
}

func TestTimeSeriesStaleWithClientUsesPRTG(t *testing.T) {
	store := &fakeChannelStore{history: testHistory(), state: staleState()}
	client := &fakePRTGClient{tsData: testPRTGTimeSeries()}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.tsCalls)
	assert.Equal(t, 0, store.historyCalls)
	assert.Contains(t, resultText(t, res), "Source: PRTG API v2 (real time)")
}

func TestCurrentValuesStaleWithClientUsesPRTG(t *testing.T) {
	store := &fakeChannelStore{
		lastValues: []types.ChannelLastValue{
			{ChannelInfo: types.ChannelInfo{ID: 1, Name: "CPU", Unit: "%"}, Value: 55.5, Timestamp: time.Now()},
		},
		state: staleState(),
	}
	client := &fakePRTGClient{chData: []prtg.Channel{{ID: "1001.1", Name: "CPU"}}}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.chCalls)
	assert.Equal(t, 0, store.lastValuesCalls)
	assert.Contains(t, resultText(t, res), "Source: PRTG API v2 (real time)")
}

func TestTimeSeriesStaleWithoutClientUsesDatabaseWithNote(t *testing.T) {
	store := &fakeChannelStore{history: testHistory(), state: staleState()}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, store.historyCalls)
	text := resultText(t, res)
	assert.Contains(t, text, "Source: PRTG Data Exporter database")
	assert.Contains(t, text, "Data Exporter stopped exporting this sensor on 2024-01-15 10:30 UTC; later values are missing.")
	assert.Contains(t, text, "Add the sensor to the Data Exporter metrics profile.")
	assert.NotContains(t, text, "use source=prtg")
}

func TestCurrentValuesStaleWithoutClientUsesDatabaseWithNote(t *testing.T) {
	store := &fakeChannelStore{
		lastValues: []types.ChannelLastValue{
			{ChannelInfo: types.ChannelInfo{ID: 1, Name: "CPU", Unit: "%"}, Value: 55.5, Timestamp: time.Now()},
		},
		state: staleState(),
	}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, store.lastValuesCalls)
	text := resultText(t, res)
	assert.Contains(t, text, "Source: PRTG Data Exporter database")
	assert.Contains(t, text, "Data Exporter stopped exporting this sensor on 2024-01-15 10:30 UTC; later values are missing.")
}

func TestTimeSeriesSourceDatabaseStaleKeepsDatabaseAndNote(t *testing.T) {
	store := &fakeChannelStore{history: testHistory(), state: staleState()}
	client := &fakePRTGClient{tsData: testPRTGTimeSeries()}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short", "source": "database"})
	assert.False(t, res.IsError)
	assert.Equal(t, 0, client.tsCalls)
	assert.Equal(t, 1, store.historyCalls)
	text := resultText(t, res)
	assert.Contains(t, text, "Source: PRTG Data Exporter database")
	assert.Contains(t, text, "Data Exporter stopped exporting this sensor on 2024-01-15 10:30 UTC")
	assert.Contains(t, text, "or use source=prtg for real-time values.")
}

func TestTimeSeriesExportStateNoChannelDataBehavesAsBefore(t *testing.T) {
	store := &fakeChannelStore{history: testHistory(), stateErr: database.ErrNoChannelData}
	client := &fakePRTGClient{tsData: testPRTGTimeSeries()}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	assert.Equal(t, 0, client.tsCalls)
	assert.Equal(t, 1, store.historyCalls)
	text := resultText(t, res)
	assert.Contains(t, text, "Source: PRTG Data Exporter database")
	assert.NotContains(t, text, "Note:")
}

func TestTimeSeriesExportStateSensorNotFoundFallsBackToPRTG(t *testing.T) {
	store := &fakeChannelStore{stateErr: database.ErrSensorNotFound}
	client := &fakePRTGClient{tsData: testPRTGTimeSeries()}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.tsCalls)
	assert.Equal(t, 0, store.historyCalls)
}

func TestTimeSeriesMissingNamesResolvedFromPRTG(t *testing.T) {
	hist := testHistory()
	hist.Channels = []types.ChannelInfo{
		{ID: 1, Name: "Channel 1", NameMissing: true},
		{ID: 2, Name: "Channel 2", NameMissing: true},
	}
	store := &fakeChannelStore{history: hist}
	client := &fakePRTGClient{chData: []prtg.Channel{
		{ID: "1001.1", Name: "Traffic In", Basic: prtg.ChannelBasic{DisplayUnit: "kbit/s"}},
		{ID: "1001.2", Name: "Traffic Out", Basic: prtg.ChannelBasic{DisplayUnit: "kbit/s"}},
	}}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.chCalls)
	text := resultText(t, res)
	assert.Contains(t, text, "(channel names from the PRTG API)")
	assert.Contains(t, text, "Traffic In (kbit/s)")
	assert.Contains(t, text, "Traffic Out (kbit/s)")
	assert.NotContains(t, text, "Channel 1")
	assert.NotContains(t, text, missingNamesNote)
}

func TestCurrentValuesMissingNamesResolvedFromPRTG(t *testing.T) {
	store := &fakeChannelStore{lastValues: []types.ChannelLastValue{
		{ChannelInfo: types.ChannelInfo{ID: 1, Name: "Channel 1", NameMissing: true}, Value: 42, Timestamp: time.Now()},
	}}
	client := &fakePRTGClient{chData: []prtg.Channel{
		{ID: "1001.1", Name: "CPU Load", Basic: prtg.ChannelBasic{DisplayUnit: "%"}},
	}}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.chCalls)
	text := resultText(t, res)
	assert.Contains(t, text, "Source: PRTG Data Exporter database (channel names from the PRTG API)")
	assert.Contains(t, text, "| CPU Load | 42.00 | % |")
}

func TestTimeSeriesMissingNamesWithoutClientKeepsFallback(t *testing.T) {
	hist := testHistory()
	hist.Channels = []types.ChannelInfo{
		{ID: 1, Name: "Channel 1", NameMissing: true},
	}
	store := &fakeChannelStore{history: hist}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	text := resultText(t, res)
	assert.Contains(t, text, "Channel 1")
	assert.Contains(t, text, missingNamesNote)
	assert.NotContains(t, text, "channel names from the PRTG API")
}

func TestExportStaleness(t *testing.T) {
	now := time.Now()
	fresh := now.Add(-time.Hour)
	stale := now.Add(-48 * time.Hour)

	s, note := exportStaleness(&types.SensorExportState{Tracked: false}, now)
	assert.True(t, s)
	assert.Contains(t, note, "does not export this sensor's channel history")

	s, note = exportStaleness(&types.SensorExportState{Tracked: true, NoMonitoringData: true}, now)
	assert.True(t, s)
	assert.Contains(t, note, "no monitoring data")

	s, note = exportStaleness(&types.SensorExportState{Tracked: true}, now)
	assert.True(t, s)
	assert.Contains(t, note, "has not exported this sensor's channel history yet")

	s, note = exportStaleness(&types.SensorExportState{Tracked: true, LastFetchUTC: &stale}, now)
	assert.True(t, s)
	assert.Contains(t, note, "stopped exporting this sensor on")

	s, note = exportStaleness(&types.SensorExportState{Tracked: true, LastFetchUTC: &fresh}, now)
	assert.False(t, s)
	assert.Empty(t, note)

	s, note = exportStaleness(nil, now)
	assert.False(t, s)
	assert.Empty(t, note)
}

func TestFormatStatisticsResponseDataExporterSections(t *testing.T) {
	next := time.Date(2024, 2, 1, 3, 0, 0, 0, time.UTC)
	stats := &types.Statistics{
		TotalSensors: 10,
		Profiles: []types.CollectionProfile{
			{
				Alias:      "metrics",
				Kind:       "metrics",
				LastEndUTC: time.Date(2024, 2, 1, 2, 0, 0, 0, time.UTC),
				NextUTC:    &next,
				Paused:     true,
				HasErrors:  true,
			},
		},
		ExportCoverage: &types.ExportCoverage{
			UpToDate:         647,
			Stale:            100,
			NeverFetched:     50,
			NoData:           20,
			NotTracked:       3380,
			FreshWithinHours: 24,
		},
	}

	out := formatStatisticsResponse(stats)
	assert.Contains(t, out, "**Data Exporter collection:**")
	assert.Contains(t, out, "- **metrics** (metrics): last collection 2024-02-01 02:00, next 2024-02-01 03:00 ⏸️ flagged paused by Data Exporter ❌ errors")
	assert.Contains(t, out, "**Channel history coverage** (fresh = exported within 24 h):")
	assert.Contains(t, out, "Up to date: 647")
	assert.Contains(t, out, "Stale: 100")
	assert.Contains(t, out, "Never fetched: 50")
	assert.Contains(t, out, "Without data: 20")
	assert.Contains(t, out, "Not tracked: 3380")
	assert.Contains(t, out, "```json")
}
