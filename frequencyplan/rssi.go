package frequencyplan

import "sort"

// RssiMin/RssiMax are the fixed RSSI scale (dBm) the dashboard bars span.
// RssiRedUntil is the inclusive top of the red zone (signal <= -90 is red),
// RssiGreenFrom the inclusive bottom of the green zone (signal >= -80 is
// green); yellow sits between them. The values are exported so the HTTP
// handler can send them to the UI, keeping the zones a single source of truth.
const (
	RssiMin       = -100
	RssiMax       = -50
	RssiRedUntil  = -90
	RssiGreenFrom = -80
)

// Client is one connected-client row (a connected_clients view record) fed to
// BuildRssiOverview. Signal 0 means "unset" and is skipped by the builder.
type Client struct {
	Device    string
	Frequency int
	Signal    int
}

// RssiMarker is one access-point mark on a band's bar: the AP's worst
// (lowest) signal among its clients on that band, plus how many of that AP's
// clients on the band reported a valid signal.
type RssiMarker struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Rssi        int    `json:"rssi"`
	ClientCount int    `json:"clientCount"`
}

// RssiBand is the rendered model for one band's bar. Min/Max/RedUntil/GreenFrom
// are echoed per band so the UI can compute zone stops and marker positions
// without hardcoding thresholds.
type RssiBand struct {
	Band      string       `json:"band"`
	Label     string       `json:"label"`
	Min       int          `json:"min"`
	Max       int          `json:"max"`
	RedUntil  int          `json:"redUntil"`
	GreenFrom int          `json:"greenFrom"`
	Markers   []RssiMarker `json:"markers"`
}

// BuildRssiOverview groups clients by band and AP, placing one marker per AP at
// the worst (lowest) signal among that AP's clients on the band. Clients with
// an unset/positive signal (Signal <= 0), no attributable device, or a band
// outside 2.4/5/6 GHz are skipped. All three bands are always emitted, in
// display order, with markers sorted by name then id for deterministic output.
func BuildRssiOverview(clients []Client, deviceNames map[string]string) []RssiBand {
	// band -> device id -> aggregated stats.
	type apStats struct {
		worst       int
		clientCount int
	}
	byBand := map[string]map[string]*apStats{}
	for _, c := range clients {
		if c.Signal >= 0 || c.Device == "" {
			continue
		}
		band := FrequencyToBand(c.Frequency)
		if !isOverviewBand(band) {
			continue
		}
		aps := byBand[band]
		if aps == nil {
			aps = map[string]*apStats{}
			byBand[band] = aps
		}
		st := aps[c.Device]
		if st == nil {
			st = &apStats{worst: c.Signal}
			aps[c.Device] = st
		} else if c.Signal < st.worst {
			st.worst = c.Signal
		}
		st.clientCount++
	}

	out := make([]RssiBand, 0, len(Bands))
	for _, band := range Bands {
		markers := make([]RssiMarker, 0, len(byBand[band]))
		for id, st := range byBand[band] {
			name := deviceNames[id]
			if name == "" {
				name = id
			}
			markers = append(markers, RssiMarker{
				Id:          id,
				Name:        name,
				Rssi:        st.worst,
				ClientCount: st.clientCount,
			})
		}
		sort.Slice(markers, func(i, j int) bool {
			if markers[i].Name != markers[j].Name {
				return markers[i].Name < markers[j].Name
			}
			return markers[i].Id < markers[j].Id
		})
		out = append(out, RssiBand{
			Band:      band,
			Label:     BandLabels[band],
			Min:       RssiMin,
			Max:       RssiMax,
			RedUntil:  RssiRedUntil,
			GreenFrom: RssiGreenFrom,
			Markers:   markers,
		})
	}
	return out
}

// isOverviewBand reports whether a band key is rendered as an RSSI bar.
func isOverviewBand(band string) bool {
	for _, b := range Bands {
		if b == band {
			return true
		}
	}
	return false
}
