<script>
    import { onMount, onDestroy } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";
    import { Chart, BarController, BarElement, CategoryScale, LinearScale, Tooltip, Legend } from "chart.js";
    import { push } from "svelte-spa-router";

    const BAND_COLORS = {
        "2.4": "#4e79a7",
        "5": "#59a14f",
        "6": "#b07aa1",
    };

    Chart.register(BarController, BarElement, CategoryScale, LinearScale, Tooltip, Legend);

    let isLoading = false;
    let bands = [];
    let chartInstances = {};

    export async function load() {
        isLoading = true;
        try {
            const res = await ApiClient.send("/api/v1/rssi-per-access-point", {
                method: "GET",
                requestKey: "rssi_per_access_point",
            });
            bands = res.bands || [];
        } catch (err) {
            if (!err?.isAbort) {
                ApiClient.error(err);
            }
        } finally {
            isLoading = false;
        }
    }

    function onSelect(bandKey, point) {
        const filter = `band = "${bandKey}" && device = "${point.device}"`;
        push(`/collections?collection=connected_clients&filter=${encodeURIComponent(filter)}`);
    }

    function barChart(node, band) {
        const color = BAND_COLORS[band.key] || "#4e79a7";
        const points = band.points || [];

        const chart = new Chart(node, {
            type: "bar",
            data: {
                labels: points.map((p) => p.name),
                datasets: [{
                    data: points.map((p) => p.rssi),
                    backgroundColor: color,
                    borderColor: color,
                    borderWidth: 1,
                    borderRadius: 3,
                }],
            },
            options: {
                indexAxis: "y",
                responsive: true,
                maintainAspectRatio: false,
                onClick: (_, elements) => {
                    if (!elements.length) return;
                    const i = elements[0].index;
                    if (i >= 0 && i < points.length) {
                        onSelect(band.key, points[i]);
                    }
                },
                plugins: {
                    legend: { display: false },
                    tooltip: {
                        callbacks: {
                            label: (ctx) => {
                                const p = points[ctx.dataIndex];
                                return p ? `${p.name}: ${p.rssi} dBm` : "";
                            },
                        },
                    },
                },
                scales: {
                    x: {
                        reverse: true,
                        title: { display: true, text: "RSSI (dBm)", color: "#617079" },
                        ticks: { color: "#617079" },
                        grid: { color: "rgba(0,0,0,0.05)" },
                    },
                    y: {
                        ticks: { color: "#617079", font: { size: 11 } },
                        grid: { display: false },
                    },
                },
            },
        });

        chartInstances[band.key] = chart;

        return {
            update(next) {
                const p = next.points || [];
                const c = BAND_COLORS[next.key] || "#4e79a7";
                chart.data.labels = p.map((pt) => pt.name);
                chart.data.datasets[0].data = p.map((pt) => pt.rssi);
                chart.data.datasets[0].backgroundColor = c;
                chart.data.datasets[0].borderColor = c;
                chart.update();
            },
            destroy() {
                chart.destroy();
                delete chartInstances[band.key];
            },
        };
    }

    onMount(() => {
        load();
    });

    onDestroy(() => {
        for (const key of Object.keys(chartInstances)) {
            chartInstances[key].destroy();
            delete chartInstances[key];
        }
    });
</script>

<div class="rssi-ap-chart" class:loading={isLoading}>
    {#if isLoading}
        <div class="chart-loader loader" transition:scale={{ duration: 150 }} />
    {/if}
    {#if !isLoading && bands.length === 0}
        <div class="rssi-empty">No RSSI data available.</div>
    {:else}
        <div class="band-charts">
            {#each bands as band (band.key)}
                <div class="band-column">
                    <div class="band-label">{band.label}</div>
                    <div class="bar-chart-wrap">
                        <canvas class="bar-chart-canvas" use:barChart={band} />
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    .rssi-ap-chart {
        position: relative;
        width: 100%;
        min-height: 120px;
    }
    .rssi-ap-chart.loading {
        opacity: 0.6;
        pointer-events: none;
    }
    .chart-loader {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        z-index: 1;
    }
    .band-charts {
        display: flex;
        gap: var(--baseSpacing);
        width: 100%;
        min-height: 260px;
    }
    .band-column {
        flex: 1;
        min-width: 0;
    }
    .band-label {
        text-align: center;
        font-size: var(--smFontSize);
        font-weight: 600;
        color: var(--txtHintColor);
        margin-bottom: 6px;
    }
    .bar-chart-wrap {
        position: relative;
        height: 240px;
    }
    .bar-chart-canvas {
        cursor: pointer;
    }
    .rssi-empty {
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
        padding: var(--baseSpacing) 0;
        text-align: center;
    }
</style>
