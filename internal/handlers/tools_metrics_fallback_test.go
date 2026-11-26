package handlers

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/matthieu/mcp-server-prtg/internal/database"
	"github.com/matthieu/mcp-server-prtg/internal/types"
)

func noDataState() *types.SensorExportState {
	return &types.SensorExportState{Tracked: true, NoMonitoringData: true}
}

func TestTimeSeriesStaleWithFailingPRTGServesDatabase(t *testing.T) {
	store := &fakeChannelStore{history: testHistory(), state: noDataState()}
	client := &fakePRTGClient{tsErr: errors.New("connection refused")}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.tsCalls)
	assert.Equal(t, 1, store.historyCalls)
	text := resultText(t, res)
	assert.Contains(t, text, "Source: PRTG Data Exporter database")
	assert.Contains(t, text, "Data Exporter reports no monitoring data for this sensor.")
	assert.Contains(t, text, "Note: the PRTG API is not reachable (Failed to fetch time series: connection refused); showing exported data.")
}

func TestTimeSeriesLiveWithFailingPRTGServesDatabaseLiveWindow(t *testing.T) {
	store := &fakeChannelStore{history: testHistory()}
	client := &fakePRTGClient{tsErr: errors.New("timeout")}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "live"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.tsCalls)
	assert.Equal(t, 1, store.historyCalls)
	assert.Equal(t, 5*time.Minute, store.lastBucket)
	assert.InDelta(t, (2 * time.Hour).Seconds(), store.lastTo.Sub(store.lastFrom).Seconds(), 2)
	text := resultText(t, res)
	assert.Contains(t, text, "Source: PRTG Data Exporter database")
	assert.Contains(t, text, "Note: the PRTG API is not reachable")
	assert.Contains(t, text, "showing exported data.")
}

func TestTimeSeriesNotFoundWithFailingPRTGReturnsPRTGError(t *testing.T) {
	store := &fakeChannelStore{stateErr: database.ErrSensorNotFound}
	client := &fakePRTGClient{tsErr: errors.New("connection refused")}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.True(t, res.IsError)
	assert.Equal(t, 1, client.tsCalls)
	assert.Contains(t, resultText(t, res), "Failed to fetch time series: connection refused")
}

func TestCurrentValuesStaleWithFailingPRTGServesDatabase(t *testing.T) {
	store := &fakeChannelStore{
		lastValues: []types.ChannelLastValue{
			{ChannelInfo: types.ChannelInfo{ID: 1, Name: "CPU", Unit: "%"}, Value: 55.5, Timestamp: time.Now()},
		},
		state: noDataState(),
	}
	client := &fakePRTGClient{chErr: errors.New("503 unavailable")}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.chCalls)
	assert.Equal(t, 1, store.lastValuesCalls)
	text := resultText(t, res)
	assert.Contains(t, text, "Source: PRTG Data Exporter database")
	assert.Contains(t, text, "| CPU | 55.50 | % |")
	assert.Contains(t, text, "Data Exporter reports no monitoring data for this sensor.")
	assert.Contains(t, text, "Note: the PRTG API is not reachable (Failed to fetch channels: 503 unavailable); showing exported data.")
}

func TestCurrentValuesNotFoundWithFailingPRTGReturnsPRTGError(t *testing.T) {
	store := &fakeChannelStore{stateErr: database.ErrSensorNotFound}
	client := &fakePRTGClient{chErr: errors.New("connection refused")}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.True(t, res.IsError)
	assert.Equal(t, 1, client.chCalls)
	assert.Contains(t, resultText(t, res), "Failed to fetch channels: connection refused")
}

func TestCurrentValuesByteUnitConverted(t *testing.T) {
	store := &fakeChannelStore{lastValues: []types.ChannelLastValue{
		{ChannelInfo: types.ChannelInfo{ID: 1, Name: "Octets libres", Unit: "Go"}, Value: 19199197184, Timestamp: time.Now()},
		{ChannelInfo: types.ChannelInfo{ID: 2, Name: "Espace libre", Unit: "%"}, Value: 30.07, Timestamp: time.Now()},
	}}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.False(t, res.IsError)
	text := resultText(t, res)
	assert.Contains(t, text, "| Octets libres | 17.88 | Go |")
	assert.Contains(t, text, "| Espace libre | 30.07 | % |")
}

func TestHistoryByteUnitConverted(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	hist := &types.ChannelHistory{
		SensorID: 1001,
		From:     now.Add(-24 * time.Hour),
		To:       now,
		Bucket:   15 * time.Minute,
		Channels: []types.ChannelInfo{
			{ID: 1, Name: "Octets libres", Unit: "Go"},
			{ID: 2, Name: "Espace libre", Unit: "%"},
		},
		Points: []types.ChannelPoint{
			{Timestamp: now, Values: map[int]float64{1: 19199197184, 2: 30.07}},
		},
	}
	store := &fakeChannelStore{history: hist}
	h := newTestMetricsHandler(nil, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	text := resultText(t, res)
	assert.Contains(t, text, "Octets libres (Go)")
	assert.Contains(t, text, "17.88")
	assert.Contains(t, text, "30.07")
}

func TestFormatAlertsResponseLimitAndBreakdown(t *testing.T) {
	alerts := make([]types.Sensor, 100)
	for i := range alerts {
		switch {
		case i < 40:
			alerts[i].Status = 5
		case i < 70:
			alerts[i].Status = 13
		default:
			alerts[i].Status = 10
		}
		alerts[i].ID = i + 1
		alerts[i].Name = "sensor"
		alerts[i].StatusText = types.GetStatusText(alerts[i].Status)
	}

	out := formatAlertsResponse(alerts)
	assert.Contains(t, out, "Showing the first **100** alerts (limit reached; filter with status or device_name)")
	assert.NotContains(t, out, "Found **100 alert(s)**")

	line5 := "- " + getStatusEmoji(5) + " **" + types.GetStatusText(5) + ":** 40 sensor(s)"
	line10 := "- " + getStatusEmoji(10) + " **" + types.GetStatusText(10) + ":** 30 sensor(s)"
	line13 := "- " + getStatusEmoji(13) + " **" + types.GetStatusText(13) + ":** 30 sensor(s)"
	assert.Contains(t, out, line5)
	assert.Contains(t, out, line10)
	assert.Contains(t, out, line13)
	assert.Contains(t, out, "Down (Acknowledged)")
	assert.Less(t, strings.Index(out, line5), strings.Index(out, line10))
	assert.Less(t, strings.Index(out, line10), strings.Index(out, line13))
}

func TestStaleSensorWithoutDataAndFailingPRTGReturnsNote(t *testing.T) {
	store := &fakeChannelStore{history: &types.ChannelHistory{}, state: staleState()}
	client := &fakePRTGClient{tsErr: errors.New("connection refused"), chErr: errors.New("connection refused")}
	h := newTestMetricsHandler(client, store)

	res := callMetricsTool(t, h, true, map[string]any{"sensor_id": 1001, "time_type": "short"})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.tsCalls)
	assert.Contains(t, resultText(t, res), "Data Exporter stopped exporting this sensor")
	assert.Contains(t, resultText(t, res), "PRTG API is not reachable")

	res = callMetricsTool(t, h, false, map[string]any{"sensor_id": 1001})
	assert.False(t, res.IsError)
	assert.Equal(t, 1, client.chCalls)
	assert.Contains(t, resultText(t, res), "PRTG API is not reachable")
}
