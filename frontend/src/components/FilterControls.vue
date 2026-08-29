<script setup>
import { statusClass } from "../utils/format.js";

defineProps({
  onlyAvailable: { type: Boolean, default: false },
  selectedCountry: { type: String, default: "" },
  selectedShadowrocket: { type: String, default: "" },
  countryOptions: { type: Array, default: () => [] },
  legend: { type: Array, default: () => [] }
});

defineEmits(["update:onlyAvailable", "update:country", "update:shadowrocket"]);
</script>

<template>
  <div class="section-controls">
    <label class="checkline">
      <input
        type="checkbox"
        :checked="onlyAvailable"
        @change="$emit('update:onlyAvailable', $event.target.checked)"
      >
      <span>只看可用</span>
    </label>
    <label class="filter-field">
      <span>地区</span>
      <select
        class="filter-select"
        :value="selectedCountry"
        @change="$emit('update:country', $event.target.value)"
      >
        <option value="">全部地区</option>
        <option v-for="option in countryOptions" :key="option.name" :value="option.name">
          {{ option.name }}（{{ option.count }}）
        </option>
      </select>
    </label>
    <label class="filter-field">
      <span>shadowrocket(小火箭)</span>
      <select
        class="filter-select"
        :value="selectedShadowrocket"
        @change="$emit('update:shadowrocket', $event.target.value)"
      >
        <option value="">全部</option>
        <option value="certain">确定有</option>
        <option value="possible">可能有</option>
        <option value="uncertain">不确定</option>
      </select>
    </label>
    <div class="legend">
      <span v-for="item in legend" :key="item.status" class="legend-item">
        <i class="dot" :class="statusClass(item.status)"></i>
        {{ item.label }}
      </span>
    </div>
  </div>
</template>
