<script setup lang="ts">
import { computed } from "vue";

// Sparkline renders a compact real-time trend line as inline SVG (CSP-safe: no
// external assets). It stretches horizontally to its container and takes its
// stroke color from the parent via `currentColor`, so each metric card can tint
// its own sparkline. Values are plotted left→right (oldest→newest) and scaled
// into [min, max]; out-of-range values are clamped.
const props = withDefaults(
  defineProps<{
    values: number[];
    max?: number;
    min?: number;
    height?: number;
  }>(),
  { max: 100, min: 0, height: 28 }
);

// Fixed plotting width; the SVG scales to its container via
// preserveAspectRatio="none", so we always plot into a 0..W space and rely on
// non-scaling strokes to keep line weight constant.
const W = 100;

const norm = computed(() => {
  const span = props.max - props.min || 1;
  return props.values.map((v) => {
    const clamped = Math.max(props.min, Math.min(props.max, v));
    return (clamped - props.min) / span; // 0..1
  });
});

const points = computed(() => {
  const n = norm.value.length;
  if (n === 0) return "";
  const h = props.height;
  if (n === 1) {
    const y = (h - norm.value[0] * h).toFixed(2);
    return `0,${y} ${W},${y}`;
  }
  const step = W / (n - 1);
  return norm.value
    .map((f, i) => `${(i * step).toFixed(2)},${(h - f * h).toFixed(2)}`)
    .join(" ");
});

// Closed polygon (line + baseline) for a soft area fill under the trend.
const area = computed(() =>
  points.value ? `0,${props.height} ${points.value} ${W},${props.height}` : ""
);

const hasData = computed(() => props.values.length > 0);
</script>

<template>
  <svg
    class="spark"
    :viewBox="`0 0 ${W} ${height}`"
    preserveAspectRatio="none"
    :style="{ height: height + 'px' }"
    aria-hidden="true"
  >
    <template v-if="hasData">
      <polygon class="spark-area" :points="area" />
      <polyline class="spark-line" :points="points" />
    </template>
    <line v-else class="spark-base" :x1="0" :y1="height - 0.5" :x2="W" :y2="height - 0.5" />
  </svg>
</template>

<style scoped>
.spark {
  width: 100%;
  display: block;
  overflow: visible;
}
.spark-line {
  fill: none;
  stroke: currentColor;
  stroke-width: 1.5;
  vector-effect: non-scaling-stroke;
  stroke-linejoin: round;
  stroke-linecap: round;
}
.spark-area {
  fill: currentColor;
  opacity: 0.12;
  stroke: none;
}
.spark-base {
  stroke: var(--border);
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}
</style>
