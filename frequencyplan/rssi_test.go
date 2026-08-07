package frequencyplan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func findRssiBand(bands []RssiBand, band string) *RssiBand {
	for i := range bands {
		if bands[i].Band == band {
			return &bands[i]
		}
	}
	return nil
}

func TestBuildRssiOverviewEmptyInput(t *testing.T) {
	bands := BuildRssiOverview(nil, nil)

	// All three bands are always emitted, in display order, with no markers.
	assert.Equal(t, []string{"2.4", "5", "6"}, []string{bands[0].Band, bands[1].Band, bands[2].Band})
	for _, b := range bands {
		assert.Equal(t, RssiMin, b.Min)
		assert.Equal(t, RssiMax, b.Max)
		assert.Equal(t, RssiRedUntil, b.RedUntil)
		assert.Equal(t, RssiGreenFrom, b.GreenFrom)
		assert.Empty(t, b.Markers)
	}
}

func TestBuildRssiOverviewGroupsAndWorstSignal(t *testing.T) {
	clients := []Client{
		{Device: "ap1", Frequency: 2412, Signal: -60}, // 2.4
		{Device: "ap1", Frequency: 2437, Signal: -85}, // 2.4, worst for ap1
		{Device: "ap1", Frequency: 5180, Signal: -75}, // 5
		{Device: "ap2", Frequency: 5200, Signal: -40}, // 5, best
		{Device: "ap2", Frequency: 5220, Signal: -95}, // 5, worst for ap2
		{Device: "ap2", Frequency: 5955, Signal: -70}, // 6
		{Device: "ap3", Frequency: 6015, Signal: -88}, // 6, worst for ap3
		{Device: "ap3", Frequency: 6055, Signal: -55}, // 6
	}
	names := map[string]string{"ap1": "AP-One", "ap2": "AP-Two", "ap3": "AP-Three"}
	bands := BuildRssiOverview(clients, names)

	b24 := findRssiBand(bands, "2.4")
	assert.Len(t, b24.Markers, 1)
	assert.Equal(t, "ap1", b24.Markers[0].Id)
	assert.Equal(t, "AP-One", b24.Markers[0].Name)
	assert.Equal(t, -85, b24.Markers[0].Rssi) // worst (lowest) of -60/-85
	assert.Equal(t, 2, b24.Markers[0].ClientCount)

	b5 := findRssiBand(bands, "5")
	assert.Len(t, b5.Markers, 2)
	assert.Equal(t, -75, rssiOf(b5, "ap1"))
	assert.Equal(t, -95, rssiOf(b5, "ap2"))
	assert.Equal(t, 1, countOf(b5, "ap1"))
	assert.Equal(t, 2, countOf(b5, "ap2"))

	b6 := findRssiBand(bands, "6")
	assert.Len(t, b6.Markers, 2)
	assert.Equal(t, -88, rssiOf(b6, "ap3"))
	assert.Equal(t, 2, countOf(b6, "ap3"))
	assert.Equal(t, -70, rssiOf(b6, "ap2"))
}

func TestBuildRssiOverviewSkipRules(t *testing.T) {
	clients := []Client{
		{Device: "ap1", Frequency: 2412, Signal: 0},    // unset signal sentinel
		{Device: "ap1", Frequency: 2437, Signal: 10},   // positive (invalid) signal
		{Device: "", Frequency: 2452, Signal: -65},     // no attributable device
		{Device: "ap1", Frequency: 1000, Signal: -70},  // unknown band
		{Device: "ap1", Frequency: 58320, Signal: -70}, // 60 GHz band (not rendered)
		{Device: "ap2", Frequency: 5200, Signal: -72},  // valid
	}
	bands := BuildRssiOverview(clients, nil)

	b24 := findRssiBand(bands, "2.4")
	assert.Empty(t, b24.Markers)

	b5 := findRssiBand(bands, "5")
	assert.Len(t, b5.Markers, 1)
	assert.Equal(t, "ap2", b5.Markers[0].Id)
	assert.Equal(t, 1, b5.Markers[0].ClientCount)

	b6 := findRssiBand(bands, "6")
	assert.Empty(t, b6.Markers)
}

func TestBuildRssiOverviewNameFallback(t *testing.T) {
	clients := []Client{{Device: "ap-unknown", Frequency: 2412, Signal: -66}}
	bands := BuildRssiOverview(clients, nil)

	m := findRssiBand(bands, "2.4").Markers[0]
	assert.Equal(t, "ap-unknown", m.Id)
	assert.Equal(t, "ap-unknown", m.Name) // unknown device -> id as name
}

func TestBuildRssiOverviewDeterministicOrder(t *testing.T) {
	// Device ids shuffled; names decide the order (then ids as tiebreak).
	clients := []Client{
		{Device: "ap-c", Frequency: 2412, Signal: -60},
		{Device: "ap-a", Frequency: 2412, Signal: -70},
		{Device: "ap-b", Frequency: 2412, Signal: -65},
	}
	names := map[string]string{"ap-a": "Zulu", "ap-b": "Alpha", "ap-c": "Mike"}
	bands := BuildRssiOverview(clients, names)

	got := make([]string, 0, 3)
	for _, m := range findRssiBand(bands, "2.4").Markers {
		got = append(got, m.Name)
	}
	assert.Equal(t, []string{"Alpha", "Mike", "Zulu"}, got)

	// Same names -> id tiebreak.
	clients2 := []Client{
		{Device: "ap-2", Frequency: 2412, Signal: -60},
		{Device: "ap-1", Frequency: 2412, Signal: -70},
	}
	names2 := map[string]string{"ap-2": "Same", "ap-1": "Same"}
	bands2 := BuildRssiOverview(clients2, names2)
	got2 := make([]string, 0, 2)
	for _, m := range findRssiBand(bands2, "2.4").Markers {
		got2 = append(got2, m.Id)
	}
	assert.Equal(t, []string{"ap-1", "ap-2"}, got2)
}

func rssiOf(b *RssiBand, id string) int {
	for _, m := range b.Markers {
		if m.Id == id {
			return m.Rssi
		}
	}
	return 0
}

func countOf(b *RssiBand, id string) int {
	for _, m := range b.Markers {
		if m.Id == id {
			return m.ClientCount
		}
	}
	return 0
}
