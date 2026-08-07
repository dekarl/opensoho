<script>
    import { onMount, onDestroy, tick } from "svelte";
    import { scale } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";

    let isLoading = false;
    let bands = []; // server-computed render model (see /api/v1/rssi-overview)

    // Custom tooltip state.
    let tip = null; // { x, y, marker }
    let tipEl;
    let hideTimer;

    // Position of a signal on the fixed scale, in percent of the bar width,
    // clamped to 0..100 so markers never overflow the track.
    function pctOf(signal, band) {
        const span = band.max - band.min;
        if (span <= 0) return 0;
        return Math.min(100, Math.max(0, ((signal - band.min) / span) * 100));
    }

    // Fill colour of a marker follows the zone its worst signal falls into.
    function zone(signal, band) {
        if (signal <= band.redUntil) return "danger";
        if (signal >= band.greenFrom) return "success";
        return "warning";
    }

    // Hard-stop gradient: danger up to redUntil, warning to greenFrom, then
    // success. Thresholds come from the backend payload (single source of truth).
    function barBackground(band) {
        const red = pctOf(band.redUntil, band);
        const green = pctOf(band.greenFrom, band);
        return (
            `linear-gradient(to right, ` +
            `var(--dangerColor) 0%, var(--dangerColor) ${red}%, ` +
            `var(--warningColor) ${red}%, var(--warningColor) ${green}%, ` +
            `var(--successColor) ${green}%, var(--successColor) 100%)`
        );
    }

    // Scale labels sit exactly on their value; edge labels are nudged inward so
    // they don't overflow the track.
    function scaleLabelStyle(value, band) {
        const p = pctOf(value, band);
        let transform = "translateX(-50%)";
        if (p <= 0) transform = "none";
        else if (p >= 100) transform = "translateX(-100%)";
        return `left:${p}%; transform:${transform}`;
    }

    async function showTip(node, marker) {
        clearTimeout(hideTimer);
        const rect = node.getBoundingClientRect();
        tip = { x: rect.left, y: rect.bottom + 6, marker };
        // Clamp to the viewport once the tooltip has rendered and we know its size.
        await tick();
        if (!tipEl || !tip) return;
        const t = tipEl.getBoundingClientRect();
        const margin = 8;
        let x = tip.x;
        let y = tip.y;
        if (x + t.width > window.innerWidth - margin) x = window.innerWidth - t.width - margin;
        if (x < margin) x = margin;
        if (y + t.height > window.innerHeight - margin) y = rect.top - t.height - 6;
        tip = { ...tip, x, y };
    }

    function scheduleHide() {
        clearTimeout(hideTimer);
        hideTimer = setTimeout(() => (tip = null), 120);
    }

    export async function load() {
        isLoading = true;
        try {
            const res = await ApiClient.send("/api/v1/rssi-overview", {
                method: "GET",
                requestKey: "rssi_overview",
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

    onMount(() => {
        load();
    });

    onDestroy(() => clearTimeout(hideTimer));
</script>

<div class="rssi-overview" class:loading={isLoading}>
    {#if isLoading}
        <div class="rssi-loader loader" transition:scale={{ duration: 150 }} />
    {/if}

    <div class="rssi-legend">
        <span class="legend-item"><span class="swatch danger" /> Weak (≤ {bands[0]?.redUntil ?? "-90"} dBm)</span>
        <span class="legend-item"><span class="swatch warning" /> Fair</span>
        <span class="legend-item"><span class="swatch success" /> Good (≥ {bands[0]?.greenFrom ?? "-80"} dBm)</span>
    </div>

    {#if bands.length === 0}
        <div class="rssi-empty">No RSSI data.</div>
    {/if}

    {#each bands as b (b.band)}
        <div class="rssi-band">
            <div class="rssi-band-label">{b.label}</div>
            <div class="rssi-bar-track" style="background:{barBackground(b)}">
                {#each b.markers as m (m.id)}
                    <div
                        class="rssi-marker {zone(m.rssi, b)}"
                        style="left:{pctOf(m.rssi, b)}%"
                        on:mouseenter={(e) => showTip(e.currentTarget, m)}
                        on:mouseleave={scheduleHide}
                    />
                {/each}
            </div>
            <div class="rssi-scale">
                <span class="scale-label" style="{scaleLabelStyle(b.min, b)}">{b.min}</span>
                <span class="scale-label" style="{scaleLabelStyle(b.redUntil, b)}">{b.redUntil}</span>
                <span class="scale-label" style="{scaleLabelStyle(b.greenFrom, b)}">{b.greenFrom}</span>
                <span class="scale-label" style="{scaleLabelStyle(b.max, b)}">{b.max}</span>
            </div>
            {#if b.markers.length === 0}
                <div class="rssi-none">No clients connected</div>
            {/if}
        </div>
    {/each}

    {#if tip}
        <div
            class="rssi-tip"
            bind:this={tipEl}
            style="left:{tip.x}px; top:{tip.y}px"
            on:mouseenter={() => clearTimeout(hideTimer)}
            on:mouseleave={scheduleHide}
        >
            <div class="tip-heading">{tip.marker.name}</div>
            <div class="tip-row">
                <span class="tip-label">RSSI</span>
                <span>{tip.marker.rssi} dBm</span>
            </div>
            <div class="tip-row">
                <span class="tip-label">Clients</span>
                <span>{tip.marker.clientCount}</span>
            </div>
        </div>
    {/if}
</div>

<style>
    .rssi-overview {
        position: relative;
        width: 100%;
        min-height: 120px;
    }
    .rssi-overview.loading {
        opacity: 0.6;
        pointer-events: none;
    }
    .rssi-loader {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        z-index: 2;
    }

    .rssi-legend {
        display: inline-flex;
        flex-wrap: wrap;
        gap: var(--smSpacing);
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
        margin-bottom: var(--smSpacing);
    }
    .legend-item {
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }
    .swatch {
        display: inline-block;
        width: 12px;
        height: 12px;
        border-radius: 2px;
    }
    .swatch.danger {
        background: var(--dangerColor);
    }
    .swatch.warning {
        background: var(--warningColor);
    }
    .swatch.success {
        background: var(--successColor);
    }

    .rssi-band {
        margin-bottom: var(--smSpacing);
    }
    .rssi-band-label {
        font-size: var(--smFontSize);
        font-weight: 600;
        color: var(--txtHintColor);
        margin-bottom: 6px;
    }
    .rssi-bar-track {
        position: relative;
        height: 14px;
        border-radius: 7px;
        border: 1px solid var(--baseAlt2Color);
        box-sizing: border-box;
    }
    .rssi-marker {
        position: absolute;
        top: 50%;
        transform: translate(-50%, -50%);
        width: 12px;
        height: 12px;
        border-radius: 50%;
        border: 2px solid var(--baseColor);
        box-shadow: 0 1px 3px var(--shadowColor);
    }
    .rssi-marker.danger {
        background: var(--dangerColor);
    }
    .rssi-marker.warning {
        background: var(--warningColor);
    }
    .rssi-marker.success {
        background: var(--successColor);
    }

    .rssi-scale {
        position: relative;
        height: 16px;
        margin-top: 4px;
    }
    .scale-label {
        position: absolute;
        font-size: 10px;
        line-height: 1;
        color: var(--txtHintColor);
        white-space: nowrap;
    }

    .rssi-none {
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
        padding-top: 2px;
    }
    .rssi-empty {
        font-size: var(--smFontSize);
        color: var(--txtHintColor);
        padding: var(--smSpacing) 0;
    }

    .rssi-tip {
        position: fixed;
        z-index: 1000;
        min-width: 160px;
        max-width: 280px;
        padding: 10px 12px;
        background: var(--baseColor);
        border: 1px solid var(--baseAlt2Color);
        border-radius: var(--baseRadius);
        box-shadow: 0 2px 10px var(--shadowColor);
        font-size: 13px;
        color: var(--txtPrimaryColor);
    }
    .tip-heading {
        font-weight: 600;
        margin-bottom: 6px;
    }
    .tip-row {
        display: flex;
        gap: 8px;
        margin-top: 4px;
        line-height: 1.4;
    }
    .tip-label {
        flex: 0 0 52px;
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        color: var(--txtHintColor);
        padding-top: 1px;
    }
</style>
